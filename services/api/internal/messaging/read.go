package messaging

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"

	"github.com/jackc/pgx/v5"
	"github.com/xynoss/relay/services/api/internal/response"
)

type ReadState struct {
	LastReadSequence int64 `json:"last_read_sequence"`
}

func decodeRead(r *http.Request, w http.ResponseWriter) (ReadState, error) {
	var input *struct {
		LastReadSequence *int64 `json:"last_read_sequence"`
	}
	state := ReadState{}
	d := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1024))
	d.DisallowUnknownFields()
	if err := d.Decode(&input); err != nil {
		return state, response.NewBadRequestError("invalid read cursor")
	}
	if input == nil || input.LastReadSequence == nil {
		return state, response.NewBadRequestError("invalid read cursor")
	}
	state.LastReadSequence = *input.LastReadSequence
	if err := d.Decode(&struct{}{}); !errors.Is(err, io.EOF) || state.LastReadSequence < 0 {
		return state, response.NewBadRequestError("invalid read cursor")
	}
	return state, nil
}

func (h *Handler) channelRead(w http.ResponseWriter, r *http.Request) { h.read(w, r, false) }
func (h *Handler) directRead(w http.ResponseWriter, r *http.Request)  { h.read(w, r, true) }

func (h *Handler) read(w http.ResponseWriter, r *http.Request, direct bool) {
	var id, actor string
	var err error
	if direct {
		id, actor, err = h.authorizeDirect(r)
	} else {
		id, actor, err = h.authorize(r)
	}
	if err != nil {
		response.RenderError(w, r, err)
		return
	}
	tx, err := h.pool.Begin(r.Context())
	if err != nil {
		response.RenderError(w, r, response.NewInternalError(err))
		return
	}
	defer tx.Rollback(r.Context())
	var conversationID string
	var head int64
	// Lock access rows through commit so removal cannot race a read acknowledgement.
	if direct {
		err = tx.QueryRow(r.Context(), `SELECT c.id,c.next_sequence-1 FROM conversations c JOIN conversation_members m ON m.conversation_id=c.id WHERE c.id=$1 AND c.kind='dm' AND m.user_id=$2 AND m.left_at IS NULL FOR SHARE OF c,m`, id, actor).Scan(&conversationID, &head)
	} else {
		err = tx.QueryRow(r.Context(), `SELECT c.id,c.next_sequence-1 FROM conversations c JOIN channels ch ON ch.id=c.channel_id JOIN workspaces w ON w.id=ch.workspace_id JOIN memberships m ON m.workspace_id=w.id WHERE ch.id=$1 AND ch.deleted_at IS NULL AND w.deleted_at IS NULL AND m.user_id=$2 AND m.state='active' AND (ch.visibility='public' OR w.owner_user_id=$2 OR EXISTS(SELECT 1 FROM channel_acl a WHERE a.channel_id=ch.id AND a.subject_type='user' AND a.subject_id=$2 AND a.capability='channel.read' AND a.effect='allow')) FOR SHARE OF c,ch,w,m`, id, actor).Scan(&conversationID, &head)
	}
	if errors.Is(err, pgx.ErrNoRows) && !direct {
		// The channel may have lost its membership after the initial check.
		// Lock and recheck it even when no conversation exists yet.
		var channelID string
		err = tx.QueryRow(r.Context(), `SELECT ch.id FROM channels ch JOIN workspaces w ON w.id=ch.workspace_id JOIN memberships m ON m.workspace_id=w.id WHERE ch.id=$1 AND ch.deleted_at IS NULL AND w.deleted_at IS NULL AND m.user_id=$2 AND m.state='active' AND (ch.visibility='public' OR w.owner_user_id=$2 OR EXISTS(SELECT 1 FROM channel_acl a WHERE a.channel_id=ch.id AND a.subject_type='user' AND a.subject_id=$2 AND a.capability='channel.read' AND a.effect='allow')) FOR SHARE OF ch,w,m`, id, actor).Scan(&channelID)
		if errors.Is(err, pgx.ErrNoRows) {
			response.RenderError(w, r, response.NewNotFoundError("channel not found"))
			return
		}
		if err != nil {
			response.RenderError(w, r, response.NewInternalError(err))
			return
		}
		if r.Method == http.MethodPost {
			state, decodeErr := decodeRead(r, w)
			if decodeErr != nil {
				response.RenderError(w, r, decodeErr)
				return
			}
			if state.LastReadSequence > 0 {
				response.RenderError(w, r, response.NewBadRequestError("read cursor exceeds conversation head"))
				return
			}
		}
		response.JSON(w, r, http.StatusOK, ReadState{})
		return
	}
	if errors.Is(err, pgx.ErrNoRows) {
		response.RenderError(w, r, response.NewNotFoundError("conversation unavailable"))
		return
	}
	if err != nil {
		response.RenderError(w, r, response.NewInternalError(err))
		return
	}
	state := ReadState{}
	if r.Method == http.MethodPost {
		state, err = decodeRead(r, w)
		if err != nil {
			response.RenderError(w, r, err)
			return
		}
		if state.LastReadSequence > head {
			response.RenderError(w, r, response.NewBadRequestError("read cursor exceeds conversation head"))
			return
		}
		err = tx.QueryRow(r.Context(), `INSERT INTO message_reads(conversation_id,user_id,last_read_sequence) VALUES($1,$2,$3) ON CONFLICT(conversation_id,user_id) DO UPDATE SET last_read_sequence=GREATEST(message_reads.last_read_sequence,EXCLUDED.last_read_sequence),updated_at=NOW() RETURNING last_read_sequence`, conversationID, actor, state.LastReadSequence).Scan(&state.LastReadSequence)
	} else {
		err = tx.QueryRow(r.Context(), `SELECT COALESCE((SELECT last_read_sequence FROM message_reads WHERE conversation_id=$1 AND user_id=$2),0)`, conversationID, actor).Scan(&state.LastReadSequence)
	}
	if err == nil {
		err = tx.Commit(r.Context())
	}
	if err != nil {
		response.RenderError(w, r, response.NewInternalError(err))
		return
	}
	response.JSON(w, r, http.StatusOK, state)
}
