package auth

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"math/big"
	"strconv"
	"strings"
	"time"
)

// Generate6DigitCode generates a cryptographically uniform 6-digit numeric OTP (100000 - 999999).
func Generate6DigitCode() (string, error) {
	// Range: [0, 900000) + 100000 -> [100000, 999999]
	max := big.NewInt(900000)
	n, err := rand.Int(rand.Reader, max)
	if err != nil {
		return "", fmt.Errorf("failed to generate secure random code: %w", err)
	}
	code := n.Int64() + 100000
	return strconv.FormatInt(code, 10), nil
}

// ComputeOTPHMAC creates an HMAC-SHA256 signature of code bound to challenge ID.
func ComputeOTPHMAC(code, challengeID, hmacKey string) string {
	mac := hmac.New(sha256.New, []byte(hmacKey))
	mac.Write([]byte(fmt.Sprintf("%s:%s", challengeID, code)))
	return hex.EncodeToString(mac.Sum(nil))
}

// VerifyOTPHMAC verifies the candidate OTP against expected HMAC in constant time.
func VerifyOTPHMAC(code, challengeID, hmacKey, expectedHMAC string) bool {
	candidate := ComputeOTPHMAC(code, challengeID, hmacKey)
	return subtle.ConstantTimeCompare([]byte(candidate), []byte(expectedHMAC)) == 1
}

// GeneratePKCE generates an RFC 7636 compliant PKCE code_verifier and code_challenge (S256).
func GeneratePKCE() (verifier, challenge string, err error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", "", fmt.Errorf("failed to generate random bytes for PKCE: %w", err)
	}

	verifier = base64.RawURLEncoding.EncodeToString(b)
	hash := sha256.Sum256([]byte(verifier))
	challenge = base64.RawURLEncoding.EncodeToString(hash[:])

	return verifier, challenge, nil
}

// GenerateOAuthState creates a signed, tamper-proof state token with a 15-minute TTL.
func GenerateOAuthState(hmacKey, returnPath string) string {
	now := time.Now().UTC().Unix()
	payload := fmt.Sprintf("%d:%s", now, returnPath)
	payloadB64 := base64.RawURLEncoding.EncodeToString([]byte(payload))

	mac := hmac.New(sha256.New, []byte(hmacKey))
	mac.Write([]byte("xynoss-oauth-state:" + payloadB64))
	sig := hex.EncodeToString(mac.Sum(nil))

	return fmt.Sprintf("%s.%s", payloadB64, sig)
}

// VerifyOAuthState validates state token signature, checks 15-minute expiration, and extracts return path.
func VerifyOAuthState(hmacKey, rawState string) (returnPath string, valid bool) {
	parts := strings.Split(rawState, ".")
	if len(parts) != 2 {
		return "", false
	}

	payloadB64 := parts[0]
	sig := parts[1]

	mac := hmac.New(sha256.New, []byte(hmacKey))
	mac.Write([]byte("xynoss-oauth-state:" + payloadB64))
	expectedSig := hex.EncodeToString(mac.Sum(nil))

	if subtle.ConstantTimeCompare([]byte(sig), []byte(expectedSig)) != 1 {
		return "", false
	}

	payloadBytes, err := base64.RawURLEncoding.DecodeString(payloadB64)
	if err != nil {
		return "", false
	}

	payloadParts := strings.SplitN(string(payloadBytes), ":", 2)
	if len(payloadParts) != 2 {
		return "", false
	}

	timestampSec, err := strconv.ParseInt(payloadParts[0], 10, 64)
	if err != nil {
		return "", false
	}

	stateAge := time.Now().UTC().Unix() - timestampSec
	if stateAge < 0 || stateAge > 900 { // 15 mins max
		return "", false
	}

	return payloadParts[1], true
}
