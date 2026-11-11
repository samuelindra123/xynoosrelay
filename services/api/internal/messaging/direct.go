package messaging

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"sort"
	"strings"
	"unicode/utf8"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/xynoss/relay/services/api/internal/authctx"
	"github.com/xynoss/relay/services/api/internal/response"
)

func (h *Handler) DirectRoutes() chi.Router {
	r := chi.NewRouter()
	r.Get("/", h.listDirect)
	r.Post("/direct", h.openDirect)
	r.Get("/{conversationID}/messages", h.listDirectMessages)
	r.Post("/{conversationID}/messages", h.sendDirectMessage)
	return r
}

type DirectConversation struct {
	ID          string `json:"id"`
	UserID      string `json:"user_id"`
	Username    string `json:"username"`
	DisplayName string `json:"display_name"`
}

func actorID(r *http.Request) (string, error) {
	a, ok := authctx.GetActor(r.Context())
	if !ok || a.UserID == "" {
		return "", response.NewUnauthorizedError("authentication required")
	}
	return a.UserID, nil
}

func (h *Handler) listDirect(w http.ResponseWriter, r *http.Request) {
	actor, err := actorID(r)
	if err != nil {
		response.RenderError(w, r, err)
		return
	}
	if h.pool == nil {
		response.RenderError(w, r, response.NewServiceUnavailableError("database unavailable"))
		return
	}
	rows, err := h.pool.Query(r.Context(), `SELECT c.id, u.id, COALESCE(u.username,''), COALESCE(NULLIF(p.display_name,''),u.username,'Pengguna')
		FROM conversations c JOIN conversation_members me ON me.conversation_id=c.id AND me.user_id=$1 AND me.left_at IS NULL
		JOIN conversation_members other ON other.conversation_id=c.id AND other.user_id<>$1 AND other.left_at IS NULL
		JOIN users u ON u.id=other.user_id AND u.deleted_at IS NULL LEFT JOIN profiles p ON p.user_id=u.id
		WHERE c.kind='dm' ORDER BY c.created_at DESC LIMIT 100`, actor)
	if err != nil {
		response.RenderError(w, r, response.NewInternalError(err))
		return
	}
	defer rows.Close()
	items := make([]DirectConversation, 0)
	for rows.Next() {
		var d DirectConversation
		if err := rows.Scan(&d.ID, &d.UserID, &d.Username, &d.DisplayName); err != nil {
			response.RenderError(w, r, response.NewInternalError(err))
			return
		}
		items = append(items, d)
	}
	if err := rows.Err(); err != nil {
		response.RenderError(w, r, response.NewInternalError(err))
		return
	}
	response.JSON(w, r, http.StatusOK, items)
}

func (h *Handler) openDirect(w http.ResponseWriter, r *http.Request) {
	actor, err := actorID(r)
	if err != nil {
		response.RenderError(w, r, err)
		return
	}
	if h.pool == nil {
		response.RenderError(w, r, response.NewServiceUnavailableError("database unavailable"))
		return
	}
	var input struct {
		UserID string `json:"user_id"`
	}
	d := json.NewDecoder(http.MaxBytesReader(w, r.Body, 2048))
	d.DisallowUnknownFields()
	if d.Decode(&input) != nil {
		response.RenderError(w, r, response.NewBadRequestError("invalid request"))
		return
	}
	if e := d.Decode(&struct{}{}); !errors.Is(e, io.EOF) {
		response.RenderError(w, r, response.NewBadRequestError("expected one JSON object"))
		return
	}
	other, err := uuid.Parse(input.UserID)
	if err != nil || other.String() == actor {
		response.RenderError(w, r, response.NewBadRequestError("invalid recipient"))
		return
	}
	tx, err := h.pool.Begin(r.Context())
	if err != nil {
		response.RenderError(w, r, response.NewInternalError(err))
		return
	}
	defer tx.Rollback(r.Context())
	var target DirectConversation
	err = tx.QueryRow(r.Context(), `SELECT u.id,COALESCE(u.username,''),COALESCE(NULLIF(p.display_name,''),u.username,'Pengguna') FROM users u LEFT JOIN profiles p ON p.user_id=u.id WHERE u.id=$1 AND u.deleted_at IS NULL`, other.String()).Scan(&target.UserID, &target.Username, &target.DisplayName)
	if errors.Is(err, pgx.ErrNoRows) {
		response.RenderError(w, r, response.NewNotFoundError("recipient not found"))
		return
	}
	if err != nil {
		response.RenderError(w, r, response.NewInternalError(err))
		return
	}
	pair := []string{actor, other.String()}
	sort.Strings(pair)
	err = tx.QueryRow(r.Context(), `INSERT INTO conversations(kind,dm_canonical_pair) VALUES('dm',$1) ON CONFLICT(dm_canonical_pair) DO UPDATE SET dm_canonical_pair=EXCLUDED.dm_canonical_pair RETURNING id`, strings.Join(pair, ":")).Scan(&target.ID)
	if err != nil {
		response.RenderError(w, r, response.NewInternalError(err))
		return
	}
	_, err = tx.Exec(r.Context(), `INSERT INTO conversation_members(conversation_id,user_id) VALUES($1,$2),($1,$3) ON CONFLICT(conversation_id,user_id) DO NOTHING`, target.ID, actor, other.String())
	if err != nil {
		response.RenderError(w, r, response.NewInternalError(err))
		return
	}
	if err = tx.Commit(r.Context()); err != nil {
		response.RenderError(w, r, response.NewInternalError(err))
		return
	}
	response.JSON(w, r, http.StatusOK, target)
}

func (h *Handler) authorizeDirect(r *http.Request) (string, string, error) {
	actor, err := actorID(r)
	if err != nil {
		return "", "", err
	}
	id := chi.URLParam(r, "conversationID")
	if _, err = uuid.Parse(id); err != nil {
		return "", "", response.NewBadRequestError("invalid conversation id")
	}
	if h.pool == nil {
		return "", "", response.NewServiceUnavailableError("database unavailable")
	}
	var allowed bool
	err = h.pool.QueryRow(r.Context(), `SELECT EXISTS(SELECT 1 FROM conversations c JOIN conversation_members m ON m.conversation_id=c.id WHERE c.id=$1 AND c.kind='dm' AND m.user_id=$2 AND m.left_at IS NULL)`, id, actor).Scan(&allowed)
	if err != nil {
		return "", "", response.NewInternalError(err)
	}
	if !allowed {
		return "", "", response.NewNotFoundError("conversation not found")
	}
	return id, actor, nil
}

func (h *Handler) listDirectMessages(w http.ResponseWriter, r *http.Request) {
	id, actor, err := h.authorizeDirect(r)
	if err != nil {
		response.RenderError(w, r, err)
		return
	}
	rows, err := h.pool.Query(r.Context(), `SELECT m.id,m.sequence,m.author_id,COALESCE(NULLIF(p.display_name,''),u.username,'Pengguna'),COALESCE(u.username,''),m.text,m.created_at FROM messages m JOIN conversation_members cm ON cm.conversation_id=m.conversation_id AND cm.user_id=$2 AND cm.left_at IS NULL JOIN users u ON u.id=m.author_id LEFT JOIN profiles p ON p.user_id=u.id WHERE m.conversation_id=$1 AND m.deleted_at IS NULL ORDER BY m.sequence DESC LIMIT 100`, id, actor)
	if err != nil {
		response.RenderError(w, r, response.NewInternalError(err))
		return
	}
	defer rows.Close()
	items := make([]Message, 0)
	for rows.Next() {
		var m Message
		if err = rows.Scan(&m.ID, &m.Sequence, &m.AuthorID, &m.Author, &m.Username, &m.Text, &m.CreatedAt); err != nil {
			response.RenderError(w, r, response.NewInternalError(err))
			return
		}
		items = append(items, m)
	}
	if err = rows.Err(); err != nil {
		response.RenderError(w, r, response.NewInternalError(err))
		return
	}
	for i, j := 0, len(items)-1; i < j; i, j = i+1, j-1 {
		items[i], items[j] = items[j], items[i]
	}
	response.JSON(w, r, http.StatusOK, items)
}

func (h *Handler) sendDirectMessage(w http.ResponseWriter, r *http.Request) {
	id, actor, err := h.authorizeDirect(r)
	if err != nil {
		response.RenderError(w, r, err)
		return
	}
	var input struct {
		Text string `json:"text"`
	}
	d := json.NewDecoder(http.MaxBytesReader(w, r.Body, 16384))
	d.DisallowUnknownFields()
	if d.Decode(&input) != nil {
		response.RenderError(w, r, response.NewBadRequestError("invalid message body"))
		return
	}
	if e := d.Decode(&struct{}{}); !errors.Is(e, io.EOF) {
		response.RenderError(w, r, response.NewBadRequestError("expected one JSON object"))
		return
	}
	input.Text = strings.TrimSpace(input.Text)
	if !utf8.ValidString(input.Text) || len([]rune(input.Text)) == 0 || len([]rune(input.Text)) > 4000 {
		response.RenderError(w, r, response.NewValidationErrorSingle("text", "message must contain 1–4000 characters"))
		return
	}
	key := strings.TrimSpace(r.Header.Get("Idempotency-Key"))
	if _, err = uuid.Parse(key); err != nil {
		response.RenderError(w, r, response.NewBadRequestError("Idempotency-Key must be a UUID"))
		return
	}
	kd := sha256.Sum256([]byte(key))
	rd := sha256.Sum256([]byte(id + "\x00" + input.Text))
	kh, rh := hex.EncodeToString(kd[:]), hex.EncodeToString(rd[:])
	tx, err := h.pool.Begin(r.Context())
	if err != nil {
		response.RenderError(w, r, response.NewInternalError(err))
		return
	}
	defer tx.Rollback(r.Context())
	var memberID string
	err = tx.QueryRow(r.Context(), `SELECT m.id FROM conversation_members m JOIN conversations c ON c.id=m.conversation_id WHERE m.conversation_id=$1 AND m.user_id=$2 AND m.left_at IS NULL AND c.kind='dm' FOR SHARE OF m`, id, actor).Scan(&memberID)
	if errors.Is(err, pgx.ErrNoRows) {
		response.RenderError(w, r, response.NewNotFoundError("conversation not found"))
		return
	}
	if err != nil {
		response.RenderError(w, r, response.NewInternalError(err))
		return
	}
	_, err = tx.Exec(r.Context(), `INSERT INTO idempotency_keys(actor_id,scope,key_hash,request_hash,status,expires_at) VALUES($1,'direct_message',$2,$3,'in_progress',NOW()+INTERVAL '24 hours') ON CONFLICT(actor_id,scope,key_hash) DO NOTHING`, actor, kh, rh)
	if err != nil {
		response.RenderError(w, r, response.NewInternalError(err))
		return
	}
	var savedHash, state string
	var body []byte
	var valid bool
	err = tx.QueryRow(r.Context(), `SELECT request_hash,status,response_body,expires_at>NOW() FROM idempotency_keys WHERE actor_id=$1 AND scope='direct_message' AND key_hash=$2 FOR UPDATE`, actor, kh).Scan(&savedHash, &state, &body, &valid)
	if err != nil {
		response.RenderError(w, r, response.NewInternalError(err))
		return
	}
	if savedHash != rh || !valid {
		response.RenderError(w, r, response.NewConflictError("idempotency key cannot be reused"))
		return
	}
	if state == "completed" {
		var m Message
		if err = json.Unmarshal(body, &m); err != nil {
			response.RenderError(w, r, response.NewInternalError(err))
			return
		}
		response.JSON(w, r, http.StatusCreated, m)
		return
	}
	if state != "in_progress" {
		response.RenderError(w, r, response.NewConflictError("request cannot be retried"))
		return
	}
	var sequence int64
	err = tx.QueryRow(r.Context(), `UPDATE conversations SET next_sequence=next_sequence+1 WHERE id=$1 RETURNING next_sequence-1`, id).Scan(&sequence)
	if err != nil {
		response.RenderError(w, r, response.NewInternalError(err))
		return
	}
	var m Message
	err = tx.QueryRow(r.Context(), `INSERT INTO messages(conversation_id,sequence,author_id,text) VALUES($1,$2,$3,$4) RETURNING id,sequence,author_id,text,created_at`, id, sequence, actor, input.Text).Scan(&m.ID, &m.Sequence, &m.AuthorID, &m.Text, &m.CreatedAt)
	if err != nil {
		response.RenderError(w, r, response.NewInternalError(err))
		return
	}
	err = tx.QueryRow(r.Context(), `SELECT COALESCE(NULLIF(p.display_name,''),u.username,'Pengguna'),COALESCE(u.username,'') FROM users u LEFT JOIN profiles p ON p.user_id=u.id WHERE u.id=$1`, actor).Scan(&m.Author, &m.Username)
	if err != nil {
		response.RenderError(w, r, response.NewInternalError(err))
		return
	}
	data, err := json.Marshal(m)
	if err != nil {
		response.RenderError(w, r, response.NewInternalError(err))
		return
	}
	_, err = tx.Exec(r.Context(), `UPDATE idempotency_keys SET status='completed',response_status=201,response_body=$1 WHERE actor_id=$2 AND scope='direct_message' AND key_hash=$3`, string(data), actor, kh)
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
