// Package messaging provides persistent messaging in public workspace channels.
package messaging

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/xynoss/relay/services/api/internal/authctx"
	"github.com/xynoss/relay/services/api/internal/response"
)

type Handler struct{ pool *pgxpool.Pool }

func NewHandler(pool *pgxpool.Pool) *Handler { return &Handler{pool: pool} }
func (h *Handler) Routes() chi.Router {
	r := chi.NewRouter()
	h.registerMessageActions(r, "channelID")
	r.Get("/{channelID}/read", h.channelRead)
	r.Post("/{channelID}/read", h.channelRead)
	r.Get("/{channelID}/access", h.listPrivateAccess)
	r.Put("/{channelID}/access", h.grantPrivateAccess)
	r.Delete("/{channelID}/access/{userID}", h.revokePrivateAccess)
	r.Get("/{channelID}/messages", h.list)
	r.Post("/{channelID}/messages", h.send)
	return r
}

type Message struct {
	ID        string    `json:"id"`
	Sequence  int64     `json:"sequence"`
	AuthorID  string    `json:"author_id"`
	Author    string    `json:"author"`
	Username  string    `json:"username"`
	Text      string    `json:"text"`
	CreatedAt time.Time `json:"created_at"`
	Revision  int       `json:"revision"`
}

func (h *Handler) authorize(r *http.Request) (string, string, error) {
	actor, ok := authctx.GetActor(r.Context())
	if !ok || actor.UserID == "" {
		return "", "", response.NewUnauthorizedError("authentication required")
	}
	id := chi.URLParam(r, "channelID")
	if _, err := uuid.Parse(id); err != nil {
		return "", "", response.NewBadRequestError("invalid channel id")
	}
	if h.pool == nil {
		return "", "", response.NewServiceUnavailableError("database unavailable")
	}
	var allowed bool
	err := h.pool.QueryRow(r.Context(), `SELECT EXISTS (
		SELECT 1 FROM channels c JOIN workspaces w ON w.id=c.workspace_id
		JOIN memberships m ON m.workspace_id=w.id AND m.user_id=$2 AND m.state='active'
		WHERE c.id=$1 AND c.deleted_at IS NULL AND w.deleted_at IS NULL AND (c.visibility='public' OR w.owner_user_id=$2 OR EXISTS(SELECT 1 FROM channel_acl a WHERE a.channel_id=c.id AND a.subject_type='user' AND a.subject_id=$2 AND a.capability='channel.read' AND a.effect='allow'))
	)`, id, actor.UserID).Scan(&allowed)
	if err != nil {
		return "", "", response.NewInternalError(err)
	}
	if !allowed {
		return "", "", response.NewNotFoundError("channel not found")
	}
	return id, actor.UserID, nil
}

func (h *Handler) list(w http.ResponseWriter, r *http.Request) {
	id, actor, err := h.authorize(r)
	if err != nil {
		response.RenderError(w, r, err)
		return
	}
	before, limit, err := historyPage(r)
	if err != nil {
		response.RenderError(w, r, err)
		return
	}
	rows, err := h.pool.Query(r.Context(), `SELECT m.id, m.sequence, m.author_id,
		COALESCE(NULLIF(p.display_name,''),u.username,'Pengguna'), COALESCE(u.username,''), m.text, m.created_at,m.revision
		FROM messages m JOIN conversations c ON c.id=m.conversation_id
		JOIN channels ch ON ch.id=c.channel_id AND ch.deleted_at IS NULL
		JOIN workspaces w ON w.id=ch.workspace_id AND w.deleted_at IS NULL
		JOIN memberships member ON member.workspace_id=w.id AND member.user_id=$2 AND member.state='active'
		JOIN users u ON u.id=m.author_id LEFT JOIN profiles p ON p.user_id=u.id
		WHERE c.channel_id=$1 AND m.deleted_at IS NULL AND ($3::bigint = 0 OR m.sequence < $3) AND (ch.visibility='public' OR w.owner_user_id=$2 OR EXISTS(SELECT 1 FROM channel_acl a WHERE a.channel_id=ch.id AND a.subject_type='user' AND a.subject_id=$2 AND a.capability='channel.read' AND a.effect='allow')) ORDER BY m.sequence DESC LIMIT $4`, id, actor, before, limit)
	if err != nil {
		response.RenderError(w, r, response.NewInternalError(err))
		return
	}
	defer rows.Close()
	items := make([]Message, 0)
	for rows.Next() {
		var m Message
		if err := rows.Scan(&m.ID, &m.Sequence, &m.AuthorID, &m.Author, &m.Username, &m.Text, &m.CreatedAt, &m.Revision); err != nil {
			response.RenderError(w, r, response.NewInternalError(err))
			return
		}
		items = append(items, m)
	}
	if err := rows.Err(); err != nil {
		response.RenderError(w, r, response.NewInternalError(err))
		return
	}
	for i, j := 0, len(items)-1; i < j; i, j = i+1, j-1 {
		items[i], items[j] = items[j], items[i]
	}
	response.JSON(w, r, http.StatusOK, items)
}

func (h *Handler) send(w http.ResponseWriter, r *http.Request) {
	id, actor, err := h.authorize(r)
	if err != nil {
		response.RenderError(w, r, err)
		return
	}
	var input struct {
		Text string `json:"text"`
	}
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 16384))
	decoder.DisallowUnknownFields()
	if decoder.Decode(&input) != nil {
		response.RenderError(w, r, response.NewBadRequestError("invalid message body"))
		return
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		response.RenderError(w, r, response.NewBadRequestError("expected one JSON object"))
		return
	}
	input.Text = strings.TrimSpace(input.Text)
	if !utf8.ValidString(input.Text) || len([]rune(input.Text)) == 0 || len([]rune(input.Text)) > 4000 {
		response.RenderError(w, r, response.NewValidationErrorSingle("text", "message must contain 1–4000 characters"))
		return
	}
	requestKey := strings.TrimSpace(r.Header.Get("Idempotency-Key"))
	if _, err := uuid.Parse(requestKey); err != nil {
		response.RenderError(w, r, response.NewBadRequestError("Idempotency-Key must be a UUID"))
		return
	}
	keyDigest := sha256.Sum256([]byte(requestKey))
	requestDigest := sha256.Sum256([]byte(id + "\x00" + input.Text))
	keyHash, requestHash := hex.EncodeToString(keyDigest[:]), hex.EncodeToString(requestDigest[:])
	tx, err := h.pool.BeginTx(r.Context(), pgx.TxOptions{})
	if err != nil {
		response.RenderError(w, r, response.NewInternalError(err))
		return
	}
	defer tx.Rollback(r.Context())
	// The membership check is repeated inside the transaction immediately before writing.
	var workspaceID string
	err = tx.QueryRow(r.Context(), `SELECT c.workspace_id FROM channels c JOIN workspaces w ON w.id=c.workspace_id JOIN memberships m ON m.workspace_id=w.id AND m.user_id=$2 AND m.state='active' WHERE c.id=$1 AND c.deleted_at IS NULL AND w.deleted_at IS NULL AND (c.visibility='public' OR w.owner_user_id=$2 OR EXISTS(SELECT 1 FROM channel_acl a WHERE a.channel_id=c.id AND a.subject_type='user' AND a.subject_id=$2 AND a.capability='channel.read' AND a.effect='allow')) FOR SHARE OF c,w,m`, id, actor).Scan(&workspaceID)
	if errors.Is(err, pgx.ErrNoRows) {
		response.RenderError(w, r, response.NewNotFoundError("channel not found"))
		return
	}
	if err != nil {
		response.RenderError(w, r, response.NewInternalError(err))
		return
	}
	// A unique key scoped to the authenticated sender serializes concurrent
	// retries. The key and message commit together or both roll back.
	_, err = tx.Exec(r.Context(), `INSERT INTO idempotency_keys (actor_id,scope,key_hash,request_hash,status,expires_at)
		VALUES ($1,'channel_message',$2,$3,'in_progress',NOW()+INTERVAL '24 hours')
		ON CONFLICT (actor_id,scope,key_hash) DO NOTHING`, actor, keyHash, requestHash)
	if err != nil {
		response.RenderError(w, r, response.NewInternalError(err))
		return
	}
	var savedHash, state string
	var savedBody []byte
	var valid bool
	err = tx.QueryRow(r.Context(), `SELECT request_hash,status,response_body,expires_at>NOW() FROM idempotency_keys
		WHERE actor_id=$1 AND scope='channel_message' AND key_hash=$2 FOR UPDATE`, actor, keyHash).Scan(&savedHash, &state, &savedBody, &valid)
	if err != nil {
		response.RenderError(w, r, response.NewInternalError(err))
		return
	}
	if savedHash != requestHash || !valid {
		response.RenderError(w, r, response.NewConflictError("idempotency key cannot be reused for this request"))
		return
	}
	if state == "completed" {
		var saved Message
		if err := json.Unmarshal(savedBody, &saved); err != nil {
			response.RenderError(w, r, response.NewInternalError(err))
			return
		}
		response.JSON(w, r, http.StatusCreated, saved)
		return
	}
	if state != "in_progress" {
		response.RenderError(w, r, response.NewConflictError("request is not available for retry"))
		return
	}
	var conversationID string
	err = tx.QueryRow(r.Context(), `INSERT INTO conversations (kind,workspace_id,channel_id) VALUES ('channel',$1,$2) ON CONFLICT (channel_id) DO UPDATE SET channel_id=EXCLUDED.channel_id RETURNING id`, workspaceID, id).Scan(&conversationID)
	if err != nil {
		response.RenderError(w, r, response.NewInternalError(err))
		return
	}
	var sequence int64
	err = tx.QueryRow(r.Context(), `UPDATE conversations SET next_sequence=next_sequence+1 WHERE id=$1 RETURNING next_sequence-1`, conversationID).Scan(&sequence)
	if err != nil {
		response.RenderError(w, r, response.NewInternalError(err))
		return
	}
	var m Message
	err = tx.QueryRow(r.Context(), `INSERT INTO messages (conversation_id,sequence,author_id,text) VALUES ($1,$2,$3,$4) RETURNING id,sequence,author_id,text,created_at`, conversationID, sequence, actor, input.Text).Scan(&m.ID, &m.Sequence, &m.AuthorID, &m.Text, &m.CreatedAt)
	if err != nil {
		response.RenderError(w, r, response.NewInternalError(err))
		return
	}
	err = tx.QueryRow(r.Context(), `SELECT COALESCE(NULLIF(p.display_name,''),u.username,'Pengguna'),COALESCE(u.username,'') FROM users u LEFT JOIN profiles p ON p.user_id=u.id WHERE u.id=$1`, actor).Scan(&m.Author, &m.Username)
	if err != nil {
		response.RenderError(w, r, response.NewInternalError(err))
		return
	}
	stored, err := json.Marshal(m)
	if err != nil {
		response.RenderError(w, r, response.NewInternalError(err))
		return
	}
	_, err = tx.Exec(r.Context(), `UPDATE idempotency_keys SET status='completed',response_status=201,response_body=$1
		WHERE actor_id=$2 AND scope='channel_message' AND key_hash=$3`, string(stored), actor, keyHash)
	if err != nil {
		response.RenderError(w, r, response.NewInternalError(err))
		return
	}
	if err = tx.Commit(r.Context()); err != nil {
		response.RenderError(w, r, response.NewInternalError(err))
		return
	}
	response.JSON(w, r, http.StatusCreated, m)
}
