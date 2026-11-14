package users

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"regexp"
	"strings"

	"github.com/xynoss/relay/services/api/internal/media"
	"github.com/xynoss/relay/services/api/internal/storage"
)

var (
	usernameRegex = regexp.MustCompile(`^[a-zA-Z0-9_]{3,32}$`)

	ErrInvalidUsername    = errors.New("username must be 3-32 characters long and contain only letters, numbers, and underscores")
	ErrInvalidDisplayName = errors.New("display name must be 1-64 characters long")
)

// UpdateProfileRequest holds incoming fields for PATCH /v1/me/profile.
type UpdateProfileRequest struct {
	DisplayName  *string `json:"display_name,omitempty"`
	Bio          *string `json:"bio,omitempty"`
	Availability *string `json:"availability,omitempty"`
	CustomStatus *string `json:"custom_status,omitempty"`
	Username     *string `json:"username,omitempty"`
	Revision     int     `json:"revision,omitempty"`
}

// CompleteOnboardingRequest payload for POST /v1/me/onboarding/complete.
type CompleteOnboardingRequest struct {
	DisplayName string  `json:"display_name"`
	Username    *string `json:"username,omitempty"`
}

// Service provides user profile operations and media attachments.
type Service struct {
	store        Store
	mediaService *media.Service
	storageProv  storage.StorageProvider
	mediaURLBase string
}

// NewService constructs a new users service instance.
func NewService(store Store, mediaService *media.Service, storageProv storage.StorageProvider, apiBaseURL ...string) *Service {
	s := &Service{
		store:        store,
		mediaService: mediaService,
		storageProv:  storageProv,
	}
	if len(apiBaseURL) > 0 {
		s.mediaURLBase = strings.TrimRight(strings.TrimSpace(apiBaseURL[0]), "/")
	}
	return s
}

// GetCurrentUser returns the standardized current user view for /v1/me.
func (s *Service) GetCurrentUser(ctx context.Context, userID string) (*CurrentUserView, error) {
	u, err := s.store.GetByID(ctx, userID)
	if err != nil {
		return nil, err
	}

	p, err := s.store.GetProfile(ctx, userID)
	if err != nil {
		return nil, err
	}

	s.resolveMediaURLs(ctx, p)

	return &CurrentUserView{
		ID:                 u.ID,
		Username:           u.Username,
		OnboardingRequired: !u.IsOnboardingCompleted(),
		Profile:            p,
	}, nil
}

// UpdateProfile validates inputs and updates the profile with optimistic concurrency control.
func (s *Service) UpdateProfile(ctx context.Context, userID string, req *UpdateProfileRequest, ifMatchRevision int) (*Profile, error) {
	current, err := s.store.GetProfile(ctx, userID)
	if err != nil {
		return nil, err
	}

	expectedRev := ifMatchRevision
	if expectedRev <= 0 && req.Revision > 0 {
		expectedRev = req.Revision
	}

	// Update fields if provided
	if req.DisplayName != nil {
		dn := strings.TrimSpace(*req.DisplayName)
		if len(dn) < 1 || len(dn) > 64 {
			return nil, ErrInvalidDisplayName
		}
		current.DisplayName = dn
	}

	if req.Bio != nil {
		current.Bio = *req.Bio
	}

	if req.Availability != nil {
		avail := strings.ToLower(strings.TrimSpace(*req.Availability))
		switch avail {
		case "online", "idle", "dnd", "offline":
			current.Availability = avail
		default:
			return nil, errors.New("availability must be one of: online, idle, dnd, offline")
		}
	}

	if req.CustomStatus != nil {
		cs := strings.TrimSpace(*req.CustomStatus)
		if len(cs) > 128 {
			return nil, errors.New("custom status cannot exceed 128 characters")
		}
		current.CustomStatus = &cs
	}

	// Optional username update
	if req.Username != nil && strings.TrimSpace(*req.Username) != "" {
		un := strings.TrimSpace(*req.Username)
		if !usernameRegex.MatchString(un) {
			return nil, ErrInvalidUsername
		}
		normalized := strings.ToLower(un)
		if err := s.store.SetUsername(ctx, userID, un, normalized); err != nil {
			return nil, err
		}
	}

	updated, err := s.store.UpdateProfile(ctx, userID, current, expectedRev)
	if err != nil {
		return nil, err
	}

	s.resolveMediaURLs(ctx, updated)
	return updated, nil
}

// SetAvatar sets a new avatar for the user after confirming media ownership.
func (s *Service) SetAvatar(ctx context.Context, userID, mediaID string) (*Profile, error) {
	// Verify media asset is ready and owned by user
	m, err := s.mediaService.GetMedia(ctx, mediaID)
	if err != nil {
		return nil, fmt.Errorf("invalid media asset: %w", err)
	}
	if m.OwnerID != userID {
		return nil, media.ErrUnauthorizedMediaAccess
	}
	if m.Status != media.StatusReady {
		return nil, errors.New("media asset must be finalized and ready before attaching to profile")
	}

	current, err := s.store.GetProfile(ctx, userID)
	if err != nil {
		return nil, err
	}

	oldMediaID := current.AvatarMediaID

	// Update avatar
	if err := s.store.SetAvatar(ctx, userID, &mediaID); err != nil {
		return nil, err
	}

	// Cleanup old avatar media if replaced
	if oldMediaID != nil && *oldMediaID != mediaID {
		_ = s.mediaService.DeleteMedia(ctx, userID, *oldMediaID)
	}

	return s.store.GetProfile(ctx, userID)
}

// RemoveAvatar clears the profile avatar and cleans up the media asset.
func (s *Service) RemoveAvatar(ctx context.Context, userID string) (*Profile, error) {
	current, err := s.store.GetProfile(ctx, userID)
	if err != nil {
		return nil, err
	}

	oldMediaID := current.AvatarMediaID
	if err := s.store.SetAvatar(ctx, userID, nil); err != nil {
		return nil, err
	}

	if oldMediaID != nil {
		_ = s.mediaService.DeleteMedia(ctx, userID, *oldMediaID)
	}

	return s.store.GetProfile(ctx, userID)
}

// SetBanner sets a new profile banner for the user.
func (s *Service) SetBanner(ctx context.Context, userID, mediaID string) (*Profile, error) {
	m, err := s.mediaService.GetMedia(ctx, mediaID)
	if err != nil {
		return nil, fmt.Errorf("invalid media asset: %w", err)
	}
	if m.OwnerID != userID {
		return nil, media.ErrUnauthorizedMediaAccess
	}
	if m.Status != media.StatusReady {
		return nil, errors.New("media asset must be finalized and ready before attaching to profile")
	}

	current, err := s.store.GetProfile(ctx, userID)
	if err != nil {
		return nil, err
	}

	oldMediaID := current.BannerMediaID

	if err := s.store.SetBanner(ctx, userID, &mediaID); err != nil {
		return nil, err
	}

	if oldMediaID != nil && *oldMediaID != mediaID {
		_ = s.mediaService.DeleteMedia(ctx, userID, *oldMediaID)
	}

	return s.store.GetProfile(ctx, userID)
}

// RemoveBanner clears the profile banner and cleans up the media asset.
func (s *Service) RemoveBanner(ctx context.Context, userID string) (*Profile, error) {
	current, err := s.store.GetProfile(ctx, userID)
	if err != nil {
		return nil, err
	}

	oldMediaID := current.BannerMediaID
	if err := s.store.SetBanner(ctx, userID, nil); err != nil {
		return nil, err
	}

	if oldMediaID != nil {
		_ = s.mediaService.DeleteMedia(ctx, userID, *oldMediaID)
	}

	return s.store.GetProfile(ctx, userID)
}

// CompleteOnboarding sets display name, optional username, and marks onboarding complete.
func (s *Service) CompleteOnboarding(ctx context.Context, userID, displayName string, username *string) (*CurrentUserView, error) {
	dn := strings.TrimSpace(displayName)
	if len(dn) < 1 || len(dn) > 64 {
		return nil, ErrInvalidDisplayName
	}

	var validUsername *string
	if username != nil && strings.TrimSpace(*username) != "" {
		un := strings.TrimSpace(*username)
		if !usernameRegex.MatchString(un) {
			return nil, ErrInvalidUsername
		}
		validUsername = &un
	}

	u, p, err := s.store.CompleteOnboarding(ctx, userID, dn, validUsername)
	if err != nil {
		return nil, err
	}

	s.resolveMediaURLs(ctx, p)

	return &CurrentUserView{
		ID:                 u.ID,
		Username:           u.Username,
		OnboardingRequired: false,
		Profile:            p,
	}, nil
}

// resolveMediaURLs populates AvatarURL and BannerURL from object storage.
func (s *Service) resolveMediaURLs(ctx context.Context, p *Profile) {
	if p.AvatarMediaID != nil {
		if m, err := s.mediaService.GetMedia(ctx, *p.AvatarMediaID); err == nil && m.Status == media.StatusReady {
			resolved := s.profileMediaURL(*p.AvatarMediaID, m.URL)
			p.AvatarURL = &resolved
		}
	}
	if p.BannerMediaID != nil {
		if m, err := s.mediaService.GetMedia(ctx, *p.BannerMediaID); err == nil && m.Status == media.StatusReady {
			resolved := s.profileMediaURL(*p.BannerMediaID, m.URL)
			p.BannerURL = &resolved
		}
	}
}

// SearchUsers searches the user directory by username, display name, or email.
func (s *Service) SearchUsers(ctx context.Context, query string, limit int) ([]*UserSearchResult, error) {
	results, err := s.store.SearchUsers(ctx, query, limit)
	if err != nil {
		return nil, err
	}
	if s.mediaService == nil {
		return results, nil
	}
	for _, result := range results {
		if result.AvatarMediaID != nil {
			if m, mediaErr := s.mediaService.GetMedia(ctx, *result.AvatarMediaID); mediaErr == nil && m.Status == media.StatusReady {
				resolved := s.profileMediaURL(*result.AvatarMediaID, m.URL)
				result.AvatarURL = &resolved
			}
		}
		if result.BannerMediaID != nil {
			if m, mediaErr := s.mediaService.GetMedia(ctx, *result.BannerMediaID); mediaErr == nil && m.Status == media.StatusReady {
				resolved := s.profileMediaURL(*result.BannerMediaID, m.URL)
				result.BannerURL = &resolved
			}
		}
	}
	return results, nil
}

func (s *Service) profileMediaURL(mediaID, fallback string) string {
	if s.mediaURLBase == "" {
		return fallback
	}
	return s.mediaURLBase + "/v1/media/profile/" + url.PathEscape(mediaID)
}
