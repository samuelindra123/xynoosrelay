package workspaces

import (
	"time"
)

type Workspace struct {
	ID          string     `json:"id"`
	Name        string     `json:"name"`
	Slug        string     `json:"slug"`
	Description string     `json:"description"`
	OwnerUserID string     `json:"owner_user_id"`
	CreatedAt   time.Time  `json:"created_at"`
	DeletedAt   *time.Time `json:"deleted_at,omitempty"`
}

type WorkspaceMember struct {
	ID          string    `json:"id"`
	WorkspaceID string    `json:"workspace_id"`
	UserID      string    `json:"user_id"`
	Role        string    `json:"role"` // "Owner", "Admin", "Moderator", "Member"
	Title       string    `json:"title"`
	JoinedAt    time.Time `json:"joined_at"`
	DisplayName string    `json:"display_name"`
	Username    string    `json:"username"`
	AvatarURL   string    `json:"avatar_url"`
}

type Channel struct {
	ID          string    `json:"id"`
	WorkspaceID string    `json:"workspace_id"`
	Slug        string    `json:"slug"`
	Name        string    `json:"name"`
	Visibility  string    `json:"visibility"`
	Position    int       `json:"position"`
	CreatedAt   time.Time `json:"created_at"`
}

type WorkspaceDetail struct {
	Workspace
	Channels []*Channel         `json:"channels"`
	Members  []*WorkspaceMember `json:"members"`
}

type Invitation struct {
	ID          string     `json:"id"`
	WorkspaceID string     `json:"workspace_id"`
	Code        string     `json:"code"`
	CreatedBy   string     `json:"created_by"`
	ExpiresAt   time.Time  `json:"expires_at"`
	MaxUses     int        `json:"max_uses"`
	Uses        int        `json:"uses"`
	RevokedAt   *time.Time `json:"revoked_at,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
}

// InviteListItem is the safe, management-facing representation of an invite.
// It deliberately never exposes the invitation secret/code after creation.
type InviteListItem struct {
	ID          string     `json:"id"`
	WorkspaceID string     `json:"workspace_id"`
	CreatedBy   string     `json:"created_by"`
	ExpiresAt   time.Time  `json:"expires_at"`
	MaxUses     int        `json:"max_uses"`
	Uses        int        `json:"uses"`
	RevokedAt   *time.Time `json:"revoked_at,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
	Status      string     `json:"status"`
}

type InvitePreview struct {
	Code                 string `json:"code"`
	WorkspaceID          string `json:"workspace_id"`
	WorkspaceName        string `json:"workspace_name"`
	WorkspaceDescription string `json:"workspace_description"`
	MemberCount          int    `json:"member_count"`
	InviterName          string `json:"inviter_name"`
	InviterAvatar        string `json:"inviter_avatar"`
	InviteURL            string `json:"invite_url"`
}
