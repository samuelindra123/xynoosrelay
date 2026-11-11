package workspaces

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrWorkspaceNotFound     = errors.New("workspace not found")
	ErrInviteNotFound        = errors.New("invitation code is invalid, expired, or revoked")
	ErrAlreadyMember         = errors.New("user is already a member of this workspace")
	ErrForbidden             = errors.New("workspace permission denied")
	ErrInvalidWorkspaceInput = errors.New("invalid workspace input")
	ErrInvalidChannelInput   = errors.New("invalid channel input")
	ErrInvalidInviteInput    = errors.New("invalid invitation input")
)

type Store interface {
	CreateWorkspace(ctx context.Context, ownerID, name, slug, description string) (*Workspace, *Channel, error)
	GetUserWorkspaces(ctx context.Context, userID string) ([]*WorkspaceDetail, error)
	GetWorkspaceByID(ctx context.Context, workspaceID string) (*WorkspaceDetail, error)
	RequireWorkspaceMembership(ctx context.Context, workspaceID, userID string) error
	RequireWorkspaceCapability(ctx context.Context, workspaceID, userID, capability string) error
	CreateChannel(ctx context.Context, workspaceID, name, slug string) (*Channel, error)
	CreateInvitation(ctx context.Context, workspaceID, userID, code string, expiresAt time.Time, maxUses int) (*Invitation, error)
	GetInvitationByCode(ctx context.Context, code string) (*Invitation, *Workspace, *InvitePreview, error)
	AcceptInvitation(ctx context.Context, code, userID string) (*WorkspaceDetail, error)
	ListInvitations(ctx context.Context, workspaceID string) ([]*InviteListItem, error)
	RevokeInvitation(ctx context.Context, workspaceID, inviteID string) error
}

type PGStore struct {
	pool *pgxpool.Pool
}

func NewPGStore(pool *pgxpool.Pool) *PGStore {
	return &PGStore{pool: pool}
}

func (s *PGStore) CreateWorkspace(ctx context.Context, ownerID, name, slug, description string) (*Workspace, *Channel, error) {
	if s.pool == nil {
		return nil, nil, errors.New("database pool is disconnected")
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	// 1. Insert Workspace
	var ws Workspace
	err = tx.QueryRow(ctx, `
		INSERT INTO workspaces (name, slug, description, owner_user_id)
		VALUES ($1, $2, $3, $4)
		RETURNING id, name, slug, description, owner_user_id, created_at, deleted_at
	`, name, slug, description, ownerID).Scan(
		&ws.ID, &ws.Name, &ws.Slug, &ws.Description, &ws.OwnerUserID, &ws.CreatedAt, &ws.DeletedAt,
	)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to insert workspace: %w", err)
	}

	// 2. Insert Membership as Owner
	_, err = tx.Exec(ctx, `
		INSERT INTO memberships (workspace_id, user_id, state, display_title)
		VALUES ($1, $2, 'active', 'Founder')
	`, ws.ID, ownerID)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to insert owner membership: %w", err)
	}

	// 3. Insert Default Channel 'umum'
	var ch Channel
	err = tx.QueryRow(ctx, `
		INSERT INTO channels (workspace_id, slug, name, position)
		VALUES ($1, 'umum', 'umum', 0)
		RETURNING id, workspace_id, slug, name, position
	`, ws.ID).Scan(
		&ch.ID, &ch.WorkspaceID, &ch.Slug, &ch.Name, &ch.Position,
	)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to insert default channel: %w", err)
	}
	ch.CreatedAt = time.Now().UTC()

	if err := tx.Commit(ctx); err != nil {
		return nil, nil, fmt.Errorf("failed to commit workspace tx: %w", err)
	}

	return &ws, &ch, nil
}

func (s *PGStore) GetUserWorkspaces(ctx context.Context, userID string) ([]*WorkspaceDetail, error) {
	if s.pool == nil {
		return nil, errors.New("database pool is disconnected")
	}

	rows, err := s.pool.Query(ctx, `
		SELECT w.id, w.name, w.slug, w.description, w.owner_user_id, w.created_at, w.deleted_at
		FROM workspaces w
		JOIN memberships m ON m.workspace_id = w.id
		WHERE m.user_id = $1 AND m.state = 'active' AND w.deleted_at IS NULL
		ORDER BY w.created_at ASC
	`, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to query user workspaces: %w", err)
	}
	defer rows.Close()

	var ids []string
	for rows.Next() {
		var ws Workspace
		if err := rows.Scan(
			&ws.ID, &ws.Name, &ws.Slug, &ws.Description, &ws.OwnerUserID, &ws.CreatedAt, &ws.DeletedAt,
		); err != nil {
			return nil, err
		}

		ids = append(ids, ws.ID)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	rows.Close() // Release this connection before loading workspace details.
	details := make([]*WorkspaceDetail, 0, len(ids))
	for _, id := range ids {
		det, err := s.GetWorkspaceByID(ctx, id)
		if err != nil {
			return nil, err
		}
		details = append(details, det)
	}

	return details, nil
}

func (s *PGStore) GetWorkspaceByID(ctx context.Context, workspaceID string) (*WorkspaceDetail, error) {
	if s.pool == nil {
		return nil, errors.New("database pool is disconnected")
	}

	var det WorkspaceDetail
	det.Channels = make([]*Channel, 0)
	det.Members = make([]*WorkspaceMember, 0)
	err := s.pool.QueryRow(ctx, `
		SELECT id, name, slug, description, owner_user_id, created_at, deleted_at
		FROM workspaces
		WHERE id = $1 AND deleted_at IS NULL
	`, workspaceID).Scan(
		&det.ID, &det.Name, &det.Slug, &det.Description, &det.OwnerUserID, &det.CreatedAt, &det.DeletedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrWorkspaceNotFound
		}
		return nil, fmt.Errorf("failed to get workspace: %w", err)
	}

	// Fetch channels
	cRows, err := s.pool.Query(ctx, `
		SELECT id, workspace_id, slug, name, position
		FROM channels
		WHERE workspace_id = $1 AND deleted_at IS NULL
		ORDER BY position ASC
	`, workspaceID)
	if err != nil {
		return nil, fmt.Errorf("load channels: %w", err)
	}
	{
		for cRows.Next() {
			var ch Channel
			if err := cRows.Scan(&ch.ID, &ch.WorkspaceID, &ch.Slug, &ch.Name, &ch.Position); err != nil {
				cRows.Close()
				return nil, err
			} else {
				ch.CreatedAt = time.Now().UTC()
				det.Channels = append(det.Channels, &ch)
			}
		}
		cRows.Close()
		if err := cRows.Err(); err != nil {
			return nil, err
		}
	}

	// Fetch members with profiles
	mRows, err := s.pool.Query(ctx, `
		SELECT m.id, m.workspace_id, m.user_id, m.display_title, m.joined_at,
		       COALESCE(p.display_name, u.username, 'Pengguna') as display_name,
		       COALESCE(u.username, 'user') as username,
		       COALESCE(ma.object_key, '') as avatar_url,
		       CASE WHEN w.owner_user_id = m.user_id THEN 'Owner' ELSE 'Member' END as role
		FROM memberships m
		JOIN workspaces w ON w.id = m.workspace_id
		JOIN users u ON u.id = m.user_id
		LEFT JOIN profiles p ON p.user_id = u.id
		LEFT JOIN media ma ON ma.id = p.avatar_media_id
		WHERE m.workspace_id = $1 AND m.state = 'active'
		ORDER BY (w.owner_user_id = m.user_id) DESC, m.joined_at ASC
	`, workspaceID)
	if err != nil {
		return nil, fmt.Errorf("load members: %w", err)
	}
	{
		defer mRows.Close()
		for mRows.Next() {
			var mem WorkspaceMember
			var title *string
			if err := mRows.Scan(
				&mem.ID, &mem.WorkspaceID, &mem.UserID, &title, &mem.JoinedAt,
				&mem.DisplayName, &mem.Username, &mem.AvatarURL, &mem.Role,
			); err != nil {
				return nil, err
			} else {
				if title != nil {
					mem.Title = *title
				}
				det.Members = append(det.Members, &mem)
			}
		}
		if err := mRows.Err(); err != nil {
			return nil, err
		}
	}

	return &det, nil
}

func (s *PGStore) RequireWorkspaceMembership(ctx context.Context, workspaceID, userID string) error {
	if s.pool == nil {
		return errors.New("database pool is disconnected")
	}
	var exists bool
	err := s.pool.QueryRow(ctx, `
		SELECT EXISTS(
			SELECT 1 FROM memberships m JOIN workspaces w ON w.id = m.workspace_id
			WHERE m.workspace_id = $1 AND m.user_id = $2 AND m.state = 'active' AND w.deleted_at IS NULL
		)
	`, workspaceID, userID).Scan(&exists)
	if err != nil {
		return fmt.Errorf("check workspace membership: %w", err)
	}
	if !exists {
		return ErrForbidden
	}
	return nil
}

// RequireWorkspaceCapability centralizes workspace authorization. Owners have
// every capability; other active members require a role capability. This makes
// it safe to add role management without scattering authorization rules.
func (s *PGStore) RequireWorkspaceCapability(ctx context.Context, workspaceID, userID, capability string) error {
	if s.pool == nil {
		return errors.New("database pool is disconnected")
	}
	var allowed bool
	err := s.pool.QueryRow(ctx, `
		SELECT EXISTS(
			SELECT 1 FROM workspaces w
			WHERE w.id = $1 AND w.deleted_at IS NULL AND w.owner_user_id = $2
			UNION ALL
			SELECT 1
			FROM memberships m
			JOIN membership_roles mr ON mr.membership_id = m.id
			JOIN roles role ON role.id = mr.role_id AND role.workspace_id = m.workspace_id
			JOIN workspaces w ON w.id = m.workspace_id AND w.deleted_at IS NULL
			JOIN role_capabilities rc ON rc.role_id = mr.role_id
			WHERE m.workspace_id = $1
			  AND m.user_id = $2
			  AND m.state = 'active'
			  AND rc.capability = $3
		)
	`, workspaceID, userID, capability).Scan(&allowed)
	if err != nil {
		return fmt.Errorf("check workspace capability: %w", err)
	}
	if !allowed {
		return ErrForbidden
	}
	return nil
}

func (s *PGStore) CreateChannel(ctx context.Context, workspaceID, name, slug string) (*Channel, error) {
	if s.pool == nil {
		return nil, errors.New("database pool is disconnected")
	}

	var ch Channel
	err := s.pool.QueryRow(ctx, `
		INSERT INTO channels (workspace_id, slug, name, position)
		VALUES ($1, $2, $3, 0)
		RETURNING id, workspace_id, slug, name, position
	`, workspaceID, slug, name).Scan(
		&ch.ID, &ch.WorkspaceID, &ch.Slug, &ch.Name, &ch.Position,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create channel: %w", err)
	}
	ch.CreatedAt = time.Now().UTC()
	return &ch, nil
}

func (s *PGStore) CreateInvitation(ctx context.Context, workspaceID, userID, code string, expiresAt time.Time, maxUses int) (*Invitation, error) {
	if s.pool == nil {
		return nil, errors.New("database pool is disconnected")
	}

	h := sha256.Sum256([]byte(code))
	secretHash := hex.EncodeToString(h[:])

	var inv Invitation
	err := s.pool.QueryRow(ctx, `
		INSERT INTO invitations (workspace_id, secret_hash, created_by, expires_at, max_uses, uses)
		VALUES ($1, $2, $3, $4, $5, 0)
		RETURNING id, workspace_id, created_by, expires_at, max_uses, uses, revoked_at, created_at
	`, workspaceID, secretHash, userID, expiresAt, maxUses).Scan(
		&inv.ID, &inv.WorkspaceID, &inv.CreatedBy, &inv.ExpiresAt, &inv.MaxUses, &inv.Uses, &inv.RevokedAt, &inv.CreatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create invitation: %w", err)
	}
	inv.Code = code
	return &inv, nil
}

func (s *PGStore) GetInvitationByCode(ctx context.Context, code string) (*Invitation, *Workspace, *InvitePreview, error) {
	if s.pool == nil {
		return nil, nil, nil, errors.New("database pool is disconnected")
	}

	var inv Invitation
	var ws Workspace
	var prev InvitePreview

	h := sha256.Sum256([]byte(code))
	secretHash := hex.EncodeToString(h[:])
	err := s.pool.QueryRow(ctx, `
		SELECT i.id, i.workspace_id, i.created_by, i.expires_at, i.max_uses, i.uses, i.revoked_at, i.created_at,
		       w.id, w.name, w.slug, w.description, w.owner_user_id, w.created_at,
		       COALESCE(p.display_name, u.username, 'Pengguna') as inviter_name,
		       COALESCE(ma.object_key, '') as inviter_avatar,
		       (SELECT COUNT(*) FROM memberships WHERE workspace_id = w.id AND state = 'active') as member_count
		FROM invitations i
		JOIN workspaces w ON w.id = i.workspace_id
		JOIN users u ON u.id = i.created_by
		LEFT JOIN profiles p ON p.user_id = u.id
		LEFT JOIN media ma ON ma.id = p.avatar_media_id
		WHERE i.secret_hash = $1 AND w.deleted_at IS NULL AND i.revoked_at IS NULL AND i.expires_at > NOW() AND (i.max_uses = 0 OR i.uses < i.max_uses)
	`, secretHash).Scan(
		&inv.ID, &inv.WorkspaceID, &inv.CreatedBy, &inv.ExpiresAt, &inv.MaxUses, &inv.Uses, &inv.RevokedAt, &inv.CreatedAt,
		&ws.ID, &ws.Name, &ws.Slug, &ws.Description, &ws.OwnerUserID, &ws.CreatedAt,
		&prev.InviterName, &prev.InviterAvatar, &prev.MemberCount,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil, nil, ErrInviteNotFound
		}
		return nil, nil, nil, fmt.Errorf("failed to query invitation: %w", err)
	}

	inv.Code = code
	prev.Code = code
	prev.WorkspaceID = ws.ID
	prev.WorkspaceName = ws.Name
	prev.WorkspaceDescription = ws.Description

	return &inv, &ws, &prev, nil
}

func (s *PGStore) AcceptInvitation(ctx context.Context, code, userID string) (*WorkspaceDetail, error) {
	if s.pool == nil {
		return nil, errors.New("database pool is disconnected")
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to begin accept tx: %w", err)
	}
	defer tx.Rollback(ctx)

	var invID, wsID string
	var maxUses, uses int
	h := sha256.Sum256([]byte(code))
	secretHash := hex.EncodeToString(h[:])
	err = tx.QueryRow(ctx, `
		SELECT id, workspace_id, max_uses, uses
		FROM invitations
		WHERE secret_hash = $1 AND revoked_at IS NULL AND expires_at > NOW()
		AND EXISTS (SELECT 1 FROM workspaces w WHERE w.id = workspace_id AND w.deleted_at IS NULL)
		FOR UPDATE
	`, secretHash).Scan(&invID, &wsID, &maxUses, &uses)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrInviteNotFound
		}
		return nil, err
	}

	// Check if already active member
	var existingCount int
	err = tx.QueryRow(ctx, `
		SELECT COUNT(*) FROM memberships WHERE workspace_id = $1 AND user_id = $2 AND state = 'active'
	`, wsID, userID).Scan(&existingCount)
	if err != nil {
		return nil, err
	}
	if existingCount == 0 && maxUses > 0 && uses >= maxUses {
		return nil, ErrInviteNotFound
	}

	if existingCount == 0 {
		// Insert active membership
		command, insertErr := tx.Exec(ctx, `
			INSERT INTO memberships (workspace_id, user_id, state, display_title)
			VALUES ($1, $2, 'active', 'Anggota')
			ON CONFLICT (workspace_id, user_id)
			DO UPDATE SET state = 'active', left_at = NULL WHERE memberships.state = 'left'
		`, wsID, userID)
		if insertErr != nil {
			return nil, fmt.Errorf("failed to insert membership: %w", insertErr)
		}
		if command.RowsAffected() == 0 {
			return nil, ErrForbidden
		}

		// Increment invitation use count
		_, err = tx.Exec(ctx, `
			UPDATE invitations SET uses = uses + 1 WHERE id = $1
		`, invID)
		if err != nil {
			return nil, fmt.Errorf("failed to increment invite uses: %w", err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("failed to commit accept tx: %w", err)
	}

	return s.GetWorkspaceByID(ctx, wsID)
}

func (s *PGStore) ListInvitations(ctx context.Context, workspaceID string) ([]*InviteListItem, error) {
	if s.pool == nil {
		return nil, errors.New("database pool is disconnected")
	}
	rows, err := s.pool.Query(ctx, `
		SELECT id, workspace_id, created_by, expires_at, max_uses, uses, revoked_at, created_at,
			CASE
				WHEN revoked_at IS NOT NULL THEN 'revoked'
				WHEN expires_at <= NOW() THEN 'expired'
				WHEN max_uses > 0 AND uses >= max_uses THEN 'exhausted'
				ELSE 'active'
			END AS status
		FROM invitations
		WHERE workspace_id = $1
		ORDER BY created_at DESC
		LIMIT 100
	`, workspaceID)
	if err != nil {
		return nil, fmt.Errorf("list invitations: %w", err)
	}
	defer rows.Close()
	items := make([]*InviteListItem, 0)
	for rows.Next() {
		item := &InviteListItem{}
		if err := rows.Scan(&item.ID, &item.WorkspaceID, &item.CreatedBy, &item.ExpiresAt, &item.MaxUses, &item.Uses, &item.RevokedAt, &item.CreatedAt, &item.Status); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (s *PGStore) RevokeInvitation(ctx context.Context, workspaceID, inviteID string) error {
	if s.pool == nil {
		return errors.New("database pool is disconnected")
	}
	command, err := s.pool.Exec(ctx, `
		UPDATE invitations SET revoked_at = COALESCE(revoked_at, NOW())
		WHERE id = $1 AND workspace_id = $2
	`, inviteID, workspaceID)
	if err != nil {
		return fmt.Errorf("revoke invitation: %w", err)
	}
	if command.RowsAffected() == 0 {
		return ErrInviteNotFound
	}
	return nil
}
