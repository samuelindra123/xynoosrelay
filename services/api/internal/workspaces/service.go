package workspaces

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"math/big"
	"regexp"
	"strings"
	"time"
)

const inviteCharset = "23456789abcdefghjkmnpqrstuvwxyzABCDEFGHJKLMNPQRSTUVWXYZ"

type Service struct {
	store       Store
	frontendURL string
}

func NewService(store Store, frontendURL string) *Service {
	return &Service{
		store:       store,
		frontendURL: frontendURL,
	}
}

// GenerateOpaqueCode generates a 12-character opaque invite secret. It is URL safe,
// avoids ambiguous glyphs, and carries enough entropy for a public capability link.
func GenerateOpaqueCode() (string, error) {
	bytes := make([]byte, 12)
	charsetLen := big.NewInt(int64(len(inviteCharset)))
	for i := range bytes {
		num, err := rand.Int(rand.Reader, charsetLen)
		if err != nil {
			return "", err
		}
		bytes[i] = inviteCharset[num.Int64()]
	}
	return string(bytes), nil
}

func Slugify(s string) string {
	reg := regexp.MustCompile("[^a-zA-Z0-9]+")
	slug := strings.ToLower(reg.ReplaceAllString(s, "-"))
	slug = strings.Trim(slug, "-")
	if len(slug) > 30 {
		slug = slug[:30]
	}
	if slug == "" {
		slug = fmt.Sprintf("ws-%d", time.Now().Unix())
	}
	return slug
}

// ResolveBaseURL only trusts application configuration. Request-controlled Origin and
// Host headers must never decide where a capability URL sends its recipient.
func (s *Service) ResolveBaseURL() string {
	if base := strings.TrimRight(strings.TrimSpace(s.frontendURL), "/"); base != "" {
		return base
	}
	return "http://localhost:3000"
}

func (s *Service) CreateWorkspace(ctx context.Context, ownerID, name, description string) (*Workspace, *Channel, error) {
	name = strings.TrimSpace(name)
	description = strings.TrimSpace(description)
	if name == "" || len([]rune(name)) > 100 {
		return nil, nil, ErrInvalidWorkspaceInput
	}
	if len([]rune(description)) > 1000 {
		return nil, nil, ErrInvalidWorkspaceInput
	}
	slug := Slugify(name)
	return s.store.CreateWorkspace(ctx, ownerID, name, slug, description)
}

func (s *Service) GetUserWorkspaces(ctx context.Context, userID string) ([]*WorkspaceDetail, error) {
	return s.store.GetUserWorkspaces(ctx, userID)
}

func (s *Service) GetWorkspaceByID(ctx context.Context, workspaceID, actorID string) (*WorkspaceDetail, error) {
	if err := s.store.RequireWorkspaceMembership(ctx, workspaceID, actorID); err != nil {
		return nil, err
	}
	return s.store.GetWorkspaceByID(ctx, workspaceID, actorID)
}

func (s *Service) CreateChannel(ctx context.Context, workspaceID, actorID, name, visibility string) (*Channel, error) {
	name = strings.TrimSpace(name)
	if name == "" || len([]rune(name)) > 50 || (visibility != "public" && visibility != "private") {
		return nil, ErrInvalidChannelInput
	}
	if err := s.store.RequireWorkspaceCapability(ctx, workspaceID, actorID, "workspace.channels.create"); err != nil {
		return nil, err
	}
	slug := Slugify(name)
	return s.store.CreateChannel(ctx, workspaceID, actorID, name, slug, visibility)
}

func (s *Service) CreateInvitation(ctx context.Context, workspaceID, userID string, duration time.Duration, maxUses int) (*Invitation, string, error) {
	if err := s.store.RequireWorkspaceCapability(ctx, workspaceID, userID, "workspace.invites.create"); err != nil {
		return nil, "", err
	}
	if duration <= 0 {
		duration = 7 * 24 * time.Hour
	}
	if duration < time.Hour || duration > 30*24*time.Hour {
		return nil, "", ErrInvalidInviteInput
	}
	if maxUses < 0 || maxUses > 10_000 {
		return nil, "", ErrInvalidInviteInput
	}

	code, err := GenerateOpaqueCode()
	if err != nil {
		return nil, "", err
	}

	expiresAt := time.Now().Add(duration)

	inv, err := s.store.CreateInvitation(ctx, workspaceID, userID, code, expiresAt, maxUses)
	if err != nil {
		return nil, "", err
	}

	baseURL := s.ResolveBaseURL()
	fullURL := fmt.Sprintf("%s/invite/%s", baseURL, code)
	return inv, fullURL, nil
}

func (s *Service) GetInvitationPreview(ctx context.Context, code string) (*InvitePreview, error) {
	code = normalizeInviteCode(code)
	if code == "" {
		return nil, ErrInviteNotFound
	}
	_, _, prev, err := s.store.GetInvitationByCode(ctx, code)
	if err != nil {
		return nil, err
	}

	baseURL := s.ResolveBaseURL()
	prev.InviteURL = fmt.Sprintf("%s/invite/%s", baseURL, code)
	return prev, nil
}

func (s *Service) AcceptInvitation(ctx context.Context, code, userID string) (*WorkspaceDetail, error) {
	code = normalizeInviteCode(code)
	if code == "" {
		return nil, ErrInviteNotFound
	}
	return s.store.AcceptInvitation(ctx, code, userID)
}

func (s *Service) ListInvitations(ctx context.Context, workspaceID, actorID string) ([]*InviteListItem, error) {
	if err := s.store.RequireWorkspaceCapability(ctx, workspaceID, actorID, "workspace.invites.manage"); err != nil {
		return nil, err
	}
	return s.store.ListInvitations(ctx, workspaceID)
}

func (s *Service) RevokeInvitation(ctx context.Context, workspaceID, inviteID, actorID string) error {
	if err := s.store.RequireWorkspaceCapability(ctx, workspaceID, actorID, "workspace.invites.manage"); err != nil {
		return err
	}
	return s.store.RevokeInvitation(ctx, workspaceID, inviteID)
}

func normalizeInviteCode(code string) string {
	code = strings.TrimSpace(code)
	if len(code) != 12 {
		return ""
	}
	for _, r := range code {
		if !strings.ContainsRune(inviteCharset, r) {
			return ""
		}
	}
	return code
}

// Keep errors stable at the service boundary so transport layers do not infer
// authorization state from a database failure.
func IsInvalidInput(err error) bool {
	return errors.Is(err, ErrInvalidWorkspaceInput) || errors.Is(err, ErrInvalidChannelInput) || errors.Is(err, ErrInvalidInviteInput)
}
