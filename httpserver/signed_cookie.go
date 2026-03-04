package httpserver

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"
)

const signedCookieDelimiter = "--"

var (
	// ErrInvalidConfig is returned when HMACCookieSigner values are invalid.
	ErrInvalidConfig = errors.New("invalid signed cookie config")

	// ErrMalformedValue is returned when a signed cookie value does not have
	// the expected wire format.
	ErrMalformedValue = errors.New("malformed signed cookie value")

	// ErrInvalidSignature is returned when signature verification fails.
	ErrInvalidSignature = errors.New("invalid signed cookie signature")

	// ErrInvalidBase64 is returned when payload base64 decoding fails.
	ErrInvalidBase64 = errors.New("invalid signed cookie payload encoding")
)

// HMACCookieSigner signs and verifies cookie payloads.
//
// This codec provides integrity (tamper detection), not confidentiality.
// Signed values can still be decoded by the client.
type HMACCookieSigner struct {
	cookieName string
	keys       [][]byte
}

// CookieSigner writes and verifies signed cookies.
type CookieSigner interface {
	Set(w http.ResponseWriter, value string, opts CookieWriteOptions) error
	Verify(r *http.Request) ([]byte, error)
}

var _ CookieSigner = HMACCookieSigner{}

// CookieWriteOptions controls attributes on cookies produced by Set.
type CookieWriteOptions struct {
	Path    string
	Domain  string
	MaxAge  int
	Expires time.Time

	// Defaults to true when nil.
	Secure *bool
	// Defaults to true when nil.
	HttpOnly *bool
	// Defaults to http.SameSiteNoneMode when nil.
	SameSite *http.SameSite
}

// NewHMACCookieSigner builds a signer with a configured cookie name and key
// ring. primaryKey is used for signing. primaryKey plus fallbackKeys are used
// for verification.
func NewHMACCookieSigner(
	cookieName string,
	primaryKey []byte,
	fallbackKeys ...[]byte,
) (HMACCookieSigner, error) {
	if cookieName == "" {
		return HMACCookieSigner{}, fmt.Errorf("%w: cookie name is required", ErrInvalidConfig)
	}
	if len(primaryKey) == 0 {
		return HMACCookieSigner{}, fmt.Errorf("%w: primary key is required", ErrInvalidConfig)
	}

	keys := make([][]byte, 1+len(fallbackKeys))
	keys[0] = primaryKey

	for i, key := range fallbackKeys {
		if len(key) == 0 {
			return HMACCookieSigner{}, fmt.Errorf("%w: fallback key %d is empty", ErrInvalidConfig, i)
		}

		keys[i+1] = key
	}

	return HMACCookieSigner{
		cookieName: cookieName,
		keys:       keys,
	}, nil
}

// Verify finds the configured cookie on the request and verifies its signed
// value.
func (s HMACCookieSigner) Verify(r *http.Request) ([]byte, error) {
	c, err := r.Cookie(s.cookieName)
	if err != nil {
		return nil, err
	}

	return s.verifySignedValue(c.Value, s.cookieName, s.keys)
}

// Set builds and writes a signed cookie to the response.
func (s HMACCookieSigner) Set(
	w http.ResponseWriter,
	value string,
	opts CookieWriteOptions,
) error {
	c := s.buildCookie(s.cookieName, s.signingKey(), value, opts)

	http.SetCookie(w, c)

	return nil
}

func (s HMACCookieSigner) buildCookie(
	name string,
	signingKey []byte,
	value string,
	opts CookieWriteOptions,
) *http.Cookie {
	signedValue := s.signValue(name, signingKey, []byte(value))

	var (
		path     = "/"
		secure   = true
		httpOnly = true
		sameSite = http.SameSiteNoneMode
	)

	if opts.Path != "" {
		path = opts.Path
	}

	if opts.Secure != nil {
		secure = *opts.Secure
	}

	if opts.HttpOnly != nil {
		httpOnly = *opts.HttpOnly
	}

	if opts.SameSite != nil {
		sameSite = *opts.SameSite
	}

	return &http.Cookie{
		Name:     name,
		Value:    signedValue,
		Path:     path,
		Domain:   opts.Domain,
		MaxAge:   opts.MaxAge,
		Expires:  opts.Expires,
		Secure:   secure,
		HttpOnly: httpOnly,
		SameSite: sameSite,
	}
}

// signValue returns "<base64 payload>--<hex hmac>".
func (s HMACCookieSigner) signValue(name string, signingKey []byte, value []byte) string {
	payload := base64.StdEncoding.EncodeToString(value)
	sig := s.sign(name, payload, signingKey)

	return payload + signedCookieDelimiter + sig
}

func (s HMACCookieSigner) verifySignedValue(
	signedValue string,
	name string,
	keys [][]byte,
) ([]byte, error) {
	parts := strings.SplitN(signedValue, signedCookieDelimiter, 2)
	if len(parts) != 2 {
		return nil, fmt.Errorf("%w: missing delimiter", ErrMalformedValue)
	}

	payload := parts[0]
	signature := parts[1]

	if signature == "" {
		return nil, fmt.Errorf("%w: signature is empty", ErrMalformedValue)
	}

	sigBytes, err := hex.DecodeString(signature)
	if err != nil {
		return nil, fmt.Errorf("%w: signature is not valid hex", ErrMalformedValue)
	}

	if len(sigBytes) != sha256.Size {
		return nil, fmt.Errorf(
			"%w: signature length %d, want %d",
			ErrMalformedValue,
			len(sigBytes),
			sha256.Size,
		)
	}

	validSignature := false
	for _, key := range keys {
		wantSigBytes := s.signBytes(name, payload, key)
		if hmac.Equal(wantSigBytes, sigBytes) {
			validSignature = true
			break
		}
	}

	if !validSignature {
		return nil, ErrInvalidSignature
	}

	raw, err := base64.StdEncoding.DecodeString(payload)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidBase64, err)
	}

	return raw, nil
}

func (s HMACCookieSigner) sign(name, payload string, key []byte) string {
	return hex.EncodeToString(s.signBytes(name, payload, key))
}

func (s HMACCookieSigner) signBytes(name, payload string, key []byte) []byte {
	mac := hmac.New(sha256.New, key)
	_, _ = mac.Write([]byte(name))
	_, _ = mac.Write([]byte{0})
	_, _ = mac.Write([]byte(payload))

	return mac.Sum(nil)
}

func (s HMACCookieSigner) signingKey() []byte {
	return s.keys[0]
}
