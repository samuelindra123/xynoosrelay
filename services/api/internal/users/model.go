package users

import (
	"time"
)

// User represents an internal user identity in XYNOSS Relay.
type User struct {
	ID                    string     `json:"id"`
	Username              *string    `json:"username,omitempty"`
	UsernameNormalized    *string    `json:"-"`
	CreatedAt             time.Time  `json:"created_at"`
	DeletedAt             *time.Time `json:"-"`
	OnboardingCompletedAt *time.Time `json:"onboarding_completed_at,omitempty"`
}

// IsOnboardingCompleted returns true if onboarding has been completed.
func (u *User) IsOnboardingCompleted() bool {
	return u.OnboardingCompletedAt != nil
}

// Profile represents the global profile details for a user.
type Profile struct {
	UserID        string    `json:"user_id"`
	DisplayName   string    `json:"display_name"`
	Bio           string    `json:"bio"`
	AvatarMediaID *string   `json:"avatar_media_id,omitempty"`
	BannerMediaID *string   `json:"banner_media_id,omitempty"`
	AvatarURL     *string   `json:"avatar_url,omitempty"`
	BannerURL     *string   `json:"banner_url,omitempty"`
	Availability  string    `json:"availability"`
	CustomStatus  *string   `json:"custom_status,omitempty"`
	Revision      int       `json:"revision"`
	UpdatedAt     time.Time `json:"updated_at"`
}

// Identity represents a linked authentication identity (Google, Email OTP, etc.).
type Identity struct {
	ID                      string    `json:"id"`
	UserID                  string    `json:"user_id"`
	Provider                string    `json:"provider"`
	ProviderSubject         string    `json:"provider_subject"`
	VerifiedEmailNormalized *string   `json:"verified_email_normalized,omitempty"`
	VerifiedAt              time.Time `json:"verified_at"`
}

// CurrentUserView represents the standardized /v1/me response payload.
type CurrentUserView struct {
	ID                 string   `json:"id"`
	Username           *string  `json:"username,omitempty"`
	OnboardingRequired bool     `json:"onboarding_required"`
	Profile            *Profile `json:"profile"`
}

// UserSearchResult represents an item in the user search / directory result.
type UserSearchResult struct {
	ID           string  `json:"id"`
	DisplayName  string  `json:"display_name"`
	Username     string  `json:"username"`
	AvatarURL    *string `json:"avatar_url,omitempty"`
	Bio          string  `json:"bio"`
	Availability string  `json:"availability"`
	CustomStatus *string `json:"custom_status,omitempty"`
}
