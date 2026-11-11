package users

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/xynoss/relay/services/api/internal/media"
	"github.com/xynoss/relay/services/api/internal/storage"
)

type mockUsersMediaStore struct {
	media map[string]*media.Media
}

func newMockUsersMediaStore() *mockUsersMediaStore {
	return &mockUsersMediaStore{media: make(map[string]*media.Media)}
}

func (m *mockUsersMediaStore) Create(ctx context.Context, item *media.Media) (*media.Media, error) {
	item.ID = "media-" + item.ObjectKey
	item.CreatedAt = time.Now()
	m.media[item.ID] = item
	return item, nil
}

func (m *mockUsersMediaStore) GetByID(ctx context.Context, id string) (*media.Media, error) {
	item, ok := m.media[id]
	if !ok || item.Status == media.StatusDeleted {
		return nil, media.ErrMediaNotFound
	}
	return item, nil
}

func (m *mockUsersMediaStore) UpdateStatus(ctx context.Context, id string, status media.MediaStatus, detectedType string, bytes int64, checksum *string) error {
	if item, ok := m.media[id]; ok {
		item.Status = status
		item.DetectedType = &detectedType
		item.Bytes = bytes
		item.Checksum = checksum
	}
	return nil
}

func (m *mockUsersMediaStore) MarkDeleted(ctx context.Context, id string) error {
	if item, ok := m.media[id]; ok {
		item.Status = media.StatusDeleted
	}
	return nil
}

type inMemoryUsersStore struct {
	users    map[string]*User
	profiles map[string]*Profile
}

func newInMemoryUsersStore() *inMemoryUsersStore {
	return &inMemoryUsersStore{
		users:    make(map[string]*User),
		profiles: make(map[string]*Profile),
	}
}

func (m *inMemoryUsersStore) CreateUserWithProfile(ctx context.Context, email, provider, subject string) (*User, *Profile, error) {
	u := &User{ID: "user-1", CreatedAt: time.Now()}
	p := &Profile{UserID: "user-1", DisplayName: "Alice", Revision: 1, UpdatedAt: time.Now()}
	m.users[u.ID] = u
	m.profiles[u.ID] = p
	return u, p, nil
}

func (m *inMemoryUsersStore) GetByID(ctx context.Context, id string) (*User, error) {
	u, ok := m.users[id]
	if !ok {
		return nil, ErrUserNotFound
	}
	return u, nil
}

func (m *inMemoryUsersStore) GetProfile(ctx context.Context, userID string) (*Profile, error) {
	p, ok := m.profiles[userID]
	if !ok {
		return nil, ErrProfileNotFound
	}
	// Return copy
	copy := *p
	return &copy, nil
}

func (m *inMemoryUsersStore) UpdateProfile(ctx context.Context, userID string, p *Profile, expectedRevision int) (*Profile, error) {
	current, ok := m.profiles[userID]
	if !ok {
		return nil, ErrProfileNotFound
	}
	if expectedRevision > 0 && current.Revision != expectedRevision {
		return nil, ErrRevisionMismatch
	}
	current.DisplayName = p.DisplayName
	current.Bio = p.Bio
	current.Availability = p.Availability
	current.CustomStatus = p.CustomStatus
	current.Revision++
	current.UpdatedAt = time.Now()
	copy := *current
	return &copy, nil
}

func (m *inMemoryUsersStore) SetUsername(ctx context.Context, userID, username, normalized string) error {
	for id, u := range m.users {
		if id != userID && u.UsernameNormalized != nil && *u.UsernameNormalized == normalized {
			return ErrUsernameConflict
		}
	}
	u := m.users[userID]
	u.Username = &username
	u.UsernameNormalized = &normalized
	return nil
}

func (m *inMemoryUsersStore) SetAvatar(ctx context.Context, userID string, mediaID *string) error {
	if p, ok := m.profiles[userID]; ok {
		p.AvatarMediaID = mediaID
		p.Revision++
	}
	return nil
}

func (m *inMemoryUsersStore) SetBanner(ctx context.Context, userID string, mediaID *string) error {
	if p, ok := m.profiles[userID]; ok {
		p.BannerMediaID = mediaID
		p.Revision++
	}
	return nil
}

func (m *inMemoryUsersStore) CompleteOnboarding(ctx context.Context, userID, displayName string, username *string) (*User, *Profile, error) {
	u := m.users[userID]
	now := time.Now()
	u.OnboardingCompletedAt = &now
	p := m.profiles[userID]
	p.DisplayName = displayName
	if username != nil {
		un := *username
		norm := strings.ToLower(un)
		u.Username = &un
		u.UsernameNormalized = &norm
	}
	return u, p, nil
}

func (m *inMemoryUsersStore) FindIdentity(ctx context.Context, provider, subject string) (*Identity, error) {
	return nil, ErrIdentityNotFound
}

func (m *inMemoryUsersStore) FindUserByEmail(ctx context.Context, emailNormalized string) (*User, error) {
	return nil, ErrUserNotFound
}

func (m *inMemoryUsersStore) LinkIdentity(ctx context.Context, userID, provider, subject, emailNormalized string) (*Identity, error) {
	return nil, nil
}

func (m *inMemoryUsersStore) SearchUsers(ctx context.Context, query string, limit int) ([]*UserSearchResult, error) {
	var results []*UserSearchResult
	for _, p := range m.profiles {
		results = append(results, &UserSearchResult{
			ID:           p.UserID,
			DisplayName:  p.DisplayName,
			Username:     "user_" + p.UserID,
			AvatarURL:    p.AvatarURL,
			Bio:          p.Bio,
			Availability: p.Availability,
		})
	}
	return results, nil
}

func TestUsers_ValidationAndConcurrency(t *testing.T) {
	userStore := newInMemoryUsersStore()
	mediaStore := newMockUsersMediaStore()
	mockStorage := storage.NewMockProvider()
	mediaSvc := media.NewService(mediaStore, mockStorage)
	userSvc := NewService(userStore, mediaSvc, mockStorage)

	ctx := context.Background()
	_, _, _ = userStore.CreateUserWithProfile(ctx, "test@example.com", "email_otp", "test@example.com")

	// 1. Invalid username: too short
	shortUN := "al"
	_, err := userSvc.UpdateProfile(ctx, "user-1", &UpdateProfileRequest{Username: &shortUN}, 0)
	if err != ErrInvalidUsername {
		t.Errorf("expected ErrInvalidUsername for short username, got %v", err)
	}

	// 2. Invalid username: special characters
	specialUN := "alice@home"
	_, err = userSvc.UpdateProfile(ctx, "user-1", &UpdateProfileRequest{Username: &specialUN}, 0)
	if err != ErrInvalidUsername {
		t.Errorf("expected ErrInvalidUsername for special char, got %v", err)
	}

	// 3. Valid username & display name
	validUN := "alice_cool"
	validDN := "Alice Wonderland"
	bio := "Hello world"
	updated, err := userSvc.UpdateProfile(ctx, "user-1", &UpdateProfileRequest{
		DisplayName: &validDN,
		Username:    &validUN,
		Bio:         &bio,
	}, 1)
	if err != nil {
		t.Fatalf("UpdateProfile failed: %v", err)
	}
	if updated.DisplayName != validDN || updated.Revision != 2 {
		t.Errorf("expected updated revision 2, got %d", updated.Revision)
	}

	// 4. Optimistic concurrency conflict: updating with stale revision 1 should fail
	_, err = userSvc.UpdateProfile(ctx, "user-1", &UpdateProfileRequest{
		DisplayName: &validDN,
	}, 1)
	if err != ErrRevisionMismatch {
		t.Errorf("expected ErrRevisionMismatch, got %v", err)
	}
}

func TestUsers_IDORMediaProtection(t *testing.T) {
	userStore := newInMemoryUsersStore()
	mediaStore := newMockUsersMediaStore()
	mockStorage := storage.NewMockProvider()
	mediaSvc := media.NewService(mediaStore, mockStorage)
	userSvc := NewService(userStore, mediaSvc, mockStorage)

	ctx := context.Background()

	// Provision User 1 and User 2
	_, _, _ = userStore.CreateUserWithProfile(ctx, "user1@example.com", "email_otp", "user1@example.com")
	u2 := &User{ID: "user-2", CreatedAt: time.Now()}
	p2 := &Profile{UserID: "user-2", DisplayName: "Bob", Revision: 1, UpdatedAt: time.Now()}
	userStore.users["user-2"] = u2
	userStore.profiles["user-2"] = p2

	// User 2 creates and finalizes a media asset
	media2, _ := mediaStore.Create(ctx, &media.Media{
		OwnerID:      "user-2",
		ObjectKey:    "avatars/bob.png",
		Status:       media.StatusReady,
		DeclaredType: "image/png",
	})

	// User 1 tries to attach User 2's media as User 1's avatar (IDOR attempt!)
	_, err := userSvc.SetAvatar(ctx, "user-1", media2.ID)
	if err != media.ErrUnauthorizedMediaAccess {
		t.Errorf("expected ErrUnauthorizedMediaAccess for IDOR avatar attach, got %v", err)
	}

	// User 1 tries to delete User 2's media
	err = mediaSvc.DeleteMedia(ctx, "user-1", media2.ID)
	if err != media.ErrUnauthorizedMediaAccess {
		t.Errorf("expected ErrUnauthorizedMediaAccess for IDOR media delete, got %v", err)
	}
}
