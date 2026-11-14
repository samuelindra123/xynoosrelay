package messaging

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/xynoss/relay/services/api/internal/authctx"
	"github.com/xynoss/relay/services/api/internal/response"
)

type messageChange struct {
	Text     string `json:"text"`
	Revision int    `json:"revision"`
}

func (h *Handler) registerMessageActions(r chi.Router, idParam string) {
	r.Patch("/{"+idParam+"}/messages/{sequence}", h.editMessage)
	r.Delete("/{"+idParam+"}/messages/{sequence}", h.deleteMessage)
	r.Put("/{"+idParam+"}/messages/{sequence}/reactions/heart", h.addHeart)
	r.Delete("/{"+idParam+"}/messages/{sequence}/reactions/heart", h.removeHeart)
}

func (h *Handler) messageTarget(r *http.Request) (conversation, actor string, sequence int64, err error) {
	if chi.URLParam(r, "channelID") != "" {
		channel, user, e := h.authorize(r)
		if e != nil {
			return "", "", 0, e
		}
		actor = user
		e = h.pool.QueryRow(r.Context(), `SELECT id FROM conversations WHERE channel_id=$1`, channel).Scan(&conversation)
		if e != nil {
			if errors.Is(e, pgx.ErrNoRows) {
				return "", "", 0, response.NewNotFoundError("message not found")
			}
			return "", "", 0, response.NewInternalError(e)
		}
	} else {
		conversation, actor, err = h.authorizeDirect(r)
		if err != nil {
			return "", "", 0, err
		}
	}
	sequence, err = strconv.ParseInt(chi.URLParam(r, "sequence"), 10, 64)
	if err != nil || sequence < 1 {
		return "", "", 0, response.NewBadRequestError("invalid message sequence")
	}
	return
}

func (h *Handler) editMessage(w http.ResponseWriter, r *http.Request) {
	conversation, actor, sequence, err := h.messageTarget(r)
	if err != nil {
		response.RenderError(w, r, err)
		return
	}
	var input messageChange
	d := json.NewDecoder(http.MaxBytesReader(w, r.Body, 8192))
	d.DisallowUnknownFields()
	if err = d.Decode(&input); err != nil {
		response.RenderError(w, r, response.NewBadRequestError("invalid edit request"))
		return
	}
	if err = d.Decode(&struct{}{}); !errors.Is(err, io.EOF) || input.Revision < 1 || !utf8.ValidString(input.Text) || len([]rune(strings.TrimSpace(input.Text))) < 1 || len([]rune(input.Text)) > 4000 {
		response.RenderError(w, r, response.NewValidationErrorSingle("text", "message and revision are invalid"))
		return
	}
	var revision int
	err = h.pool.QueryRow(r.Context(), `UPDATE messages SET text=$1,revision=revision+1,edited_at=NOW() WHERE conversation_id=$2 AND sequence=$3 AND author_id=$4 AND deleted_at IS NULL AND revision=$5 RETURNING revision`, strings.TrimSpace(input.Text), conversation, sequence, actor, input.Revision).Scan(&revision)
	if errors.Is(err, pgx.ErrNoRows) {
		response.RenderError(w, r, response.NewConflictError("message changed, was deleted, or is not yours"))
		return
	}
	if err != nil {
		response.RenderError(w, r, response.NewInternalError(err))
		return
	}
	response.JSON(w, r, http.StatusOK, map[string]any{"sequence": sequence, "text": strings.TrimSpace(input.Text), "revision": revision})
}

func (h *Handler) deleteMessage(w http.ResponseWriter, r *http.Request) {
	conversation, actor, sequence, err := h.messageTarget(r)
	if err != nil {
		response.RenderError(w, r, err)
		return
	}
	_, err = h.pool.Exec(r.Context(), `UPDATE messages SET text='',deleted_at=NOW(),revision=revision+1 WHERE conversation_id=$1 AND sequence=$2 AND author_id=$3 AND deleted_at IS NULL`, conversation, sequence, actor)
	if err != nil {
		response.RenderError(w, r, response.NewInternalError(err))
		return
	}
	response.NoContent(w)
}

func (h *Handler) react(w http.ResponseWriter, r *http.Request, add bool) {
	conversation, actor, sequence, err := h.messageTarget(r)
	if err != nil {
		response.RenderError(w, r, err)
		return
	}
	var messageID string
	err = h.pool.QueryRow(r.Context(), `SELECT id FROM messages WHERE conversation_id=$1 AND sequence=$2 AND deleted_at IS NULL`, conversation, sequence).Scan(&messageID)
	if errors.Is(err, pgx.ErrNoRows) {
		response.RenderError(w, r, response.NewNotFoundError("message not found"))
		return
	}
	if err != nil {
		response.RenderError(w, r, response.NewInternalError(err))
		return
	}
	if add {
		_, err = h.pool.Exec(r.Context(), `INSERT INTO reactions(message_id,user_id,emoji) VALUES($1,$2,'❤️') ON CONFLICT(message_id,user_id,emoji) DO NOTHING`, messageID, actor)
	} else {
		_, err = h.pool.Exec(r.Context(), `DELETE FROM reactions WHERE message_id=$1 AND user_id=$2 AND emoji='❤️'`, messageID, actor)
	}
	if err != nil {
		response.RenderError(w, r, response.NewInternalError(err))
		return
	}
	response.NoContent(w)
}
func (h *Handler) addHeart(w http.ResponseWriter, r *http.Request)    { h.react(w, r, true) }
func (h *Handler) removeHeart(w http.ResponseWriter, r *http.Request) { h.react(w, r, false) }

func (h *Handler) privateChannelOwner(r *http.Request) (string, error) {
	actor, ok := authctx.GetActor(r.Context())
	if !ok || actor.UserID == "" {
		return "", response.NewUnauthorizedError("authentication required")
	}
	channelID := chi.URLParam(r, "channelID")
	if _, err := uuid.Parse(channelID); err != nil {
		return "", response.NewBadRequestError("invalid channel id")
	}
	if h.pool == nil {
		return "", response.NewServiceUnavailableError("database unavailable")
	}
	var isOwner bool
	err := h.pool.QueryRow(r.Context(), `SELECT EXISTS(SELECT 1 FROM channels c JOIN workspaces w ON w.id=c.workspace_id JOIN memberships m ON m.workspace_id=w.id AND m.user_id=$2 AND m.state='active' WHERE c.id=$1 AND c.visibility='private' AND c.deleted_at IS NULL AND w.deleted_at IS NULL AND w.owner_user_id=$2)`, channelID, actor.UserID).Scan(&isOwner)
	if err != nil {
		return "", response.NewInternalError(err)
	}
	if !isOwner {
		return "", response.NewNotFoundError("private channel not found")
	}
	return actor.UserID, nil
}

func (h *Handler) listPrivateAccess(w http.ResponseWriter, r *http.Request) {
	if _, err := h.privateChannelOwner(r); err != nil {
		response.RenderError(w, r, err)
		return
	}
	rows, err := h.pool.Query(r.Context(), `SELECT subject_id FROM channel_acl WHERE channel_id=$1 AND subject_type='user' AND capability='channel.read' AND effect='allow' ORDER BY subject_id`, chi.URLParam(r, "channelID"))
	if err != nil {
		response.RenderError(w, r, response.NewInternalError(err))
		return
	}
	defer rows.Close()
	ids := make([]string, 0)
	for rows.Next() {
		var id string
		if err = rows.Scan(&id); err != nil {
			response.RenderError(w, r, response.NewInternalError(err))
			return
		}
		ids = append(ids, id)
	}
	if err = rows.Err(); err != nil {
		response.RenderError(w, r, response.NewInternalError(err))
		return
	}
	response.JSON(w, r, http.StatusOK, map[string]any{"user_ids": ids})
}

func (h *Handler) grantPrivateAccess(w http.ResponseWriter, r *http.Request) {
	if _, err := h.privateChannelOwner(r); err != nil {
		response.RenderError(w, r, err)
		return
	}
	var input struct {
		UserID string `json:"user_id"`
	}
	d := json.NewDecoder(http.MaxBytesReader(w, r.Body, 2048))
	d.DisallowUnknownFields()
	if err := d.Decode(&input); err != nil {
		response.RenderError(w, r, response.NewBadRequestError("invalid user id"))
		return
	}
	userID, err := uuid.Parse(input.UserID)
	if err != nil {
		response.RenderError(w, r, response.NewBadRequestError("invalid user id"))
		return
	}
	tx, err := h.pool.Begin(r.Context())
	if err != nil {
		response.RenderError(w, r, response.NewInternalError(err))
		return
	}
	defer tx.Rollback(r.Context())
	cmd, err := tx.Exec(r.Context(), `INSERT INTO channel_acl(channel_id,subject_type,subject_id,capability,effect) SELECT $1,'user',$2,'channel.read','allow' FROM channels c JOIN memberships m ON m.workspace_id=c.workspace_id AND m.user_id=$2 AND m.state='active' WHERE c.id=$1 AND c.visibility='private' ON CONFLICT(channel_id,subject_type,subject_id,capability) DO UPDATE SET effect='allow'`, chi.URLParam(r, "channelID"), userID.String())
	if err != nil {
		response.RenderError(w, r, response.NewInternalError(err))
		return
	}
	if cmd.RowsAffected() == 0 {
		response.RenderError(w, r, response.NewNotFoundError("active workspace member not found"))
		return
	}
	if err = tx.Commit(r.Context()); err != nil {
		response.RenderError(w, r, response.NewInternalError(err))
		return
	}
	response.NoContent(w)
}

func (h *Handler) revokePrivateAccess(w http.ResponseWriter, r *http.Request) {
	if _, err := h.privateChannelOwner(r); err != nil {
		response.RenderError(w, r, err)
		return
	}
	userID, err := uuid.Parse(chi.URLParam(r, "userID"))
	if err != nil {
		response.RenderError(w, r, response.NewBadRequestError("invalid user id"))
		return
	}
	_, err = h.pool.Exec(r.Context(), `DELETE FROM channel_acl WHERE channel_id=$1 AND subject_type='user' AND subject_id=$2 AND capability='channel.read'`, chi.URLParam(r, "channelID"), userID.String())
	if err != nil {
		response.RenderError(w, r, response.NewInternalError(err))
		return
	}
	response.NoContent(w)
}
