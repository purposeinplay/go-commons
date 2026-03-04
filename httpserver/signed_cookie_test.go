package httpserver_test

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/purposeinplay/go-commons/httpserver"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHMACCookieSigner_SetVerifyRoundTrip(t *testing.T) {
	t.Parallel()

	signer := mustSigner(t, "session_id", primaryKey())
	c := mustSetCookie(t, signer, "hello world", httpserver.CookieWriteOptions{})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.AddCookie(c)

	got, err := signer.Verify(req)
	require.NoError(t, err)
	assert.Equal(t, []byte("hello world"), got)
}

func TestHMACCookieSigner_SetVerify_EmptyPayload(t *testing.T) {
	t.Parallel()

	signer := mustSigner(t, "session_id", primaryKey())
	c := mustSetCookie(t, signer, "", httpserver.CookieWriteOptions{})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.AddCookie(c)

	got, err := signer.Verify(req)
	require.NoError(t, err)
	assert.Equal(t, []byte(""), got)
}

func TestHMACCookieSigner_TamperDetection(t *testing.T) {
	t.Parallel()

	signer := mustSigner(t, "session_id", primaryKey())
	c := mustSetCookie(t, signer, "hello", httpserver.CookieWriteOptions{})

	parts := strings.SplitN(c.Value, "--", 2)
	require.Len(t, parts, 2)

	t.Run("tampered_payload", func(t *testing.T) {
		t.Parallel()

		tamperedCookie := *c
		tamperedCookie.Value = "X" + parts[0][1:] + "--" + parts[1]

		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.AddCookie(&tamperedCookie)

		_, err := signer.Verify(req)
		require.Error(t, err)
		assert.ErrorIs(t, err, httpserver.ErrInvalidSignature)
	})

	t.Run("tampered_signature", func(t *testing.T) {
		t.Parallel()

		tamperedCookie := *c
		tamperedCookie.Value = parts[0] + "--" + mutateHexDigit(parts[1])

		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.AddCookie(&tamperedCookie)

		_, err := signer.Verify(req)
		require.Error(t, err)
		assert.ErrorIs(t, err, httpserver.ErrInvalidSignature)
	})
}

func TestHMACCookieSigner_NameBinding(t *testing.T) {
	t.Parallel()

	key := primaryKey()
	signerA := mustSigner(t, "cookie_a", key)
	signerB := mustSigner(t, "cookie_b", key)

	c := mustSetCookie(t, signerA, "hello", httpserver.CookieWriteOptions{})

	// Replay the signed value under a different cookie name.
	rebound := &http.Cookie{Name: "cookie_b", Value: c.Value}
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.AddCookie(rebound)

	_, err := signerB.Verify(req)
	require.Error(t, err)
	assert.ErrorIs(t, err, httpserver.ErrInvalidSignature)
}

func TestHMACCookieSigner_KeyRotation(t *testing.T) {
	t.Parallel()

	oldKey := []byte("old-key-32-bytes-0000000000000000")
	newKey := []byte("new-key-32-bytes-1111111111111111")

	oldSigner := mustSigner(t, "session_id", oldKey)
	// First key signs, all keys verify.
	newSigner := mustSigner(t, "session_id", newKey, oldKey)

	oldCookie := mustSetCookie(t, oldSigner, "legacy", httpserver.CookieWriteOptions{})
	oldReq := httptest.NewRequest(http.MethodGet, "/", nil)
	oldReq.AddCookie(oldCookie)

	got, err := newSigner.Verify(oldReq)
	require.NoError(t, err)
	assert.Equal(t, []byte("legacy"), got)

	newCookie := mustSetCookie(t, newSigner, "fresh", httpserver.CookieWriteOptions{})
	newReq := httptest.NewRequest(http.MethodGet, "/", nil)
	newReq.AddCookie(newCookie)

	_, err = oldSigner.Verify(newReq)
	require.Error(t, err)
	assert.ErrorIs(t, err, httpserver.ErrInvalidSignature)
}

func TestHMACCookieSigner_MalformedAndEncodingErrors(t *testing.T) {
	t.Parallel()

	signer := mustSigner(t, "session_id", primaryKey())

	t.Run("missing_delimiter", func(t *testing.T) {
		t.Parallel()
		req := requestWithCookie("session_id", "abcdef")
		_, err := signer.Verify(req)
		require.Error(t, err)
		assert.ErrorIs(t, err, httpserver.ErrMalformedValue)
	})

	t.Run("signature_not_hex", func(t *testing.T) {
		t.Parallel()
		req := requestWithCookie("session_id", "YWJj--thisisnothex")
		_, err := signer.Verify(req)
		require.Error(t, err)
		assert.ErrorIs(t, err, httpserver.ErrMalformedValue)
	})

	t.Run("signature_wrong_length", func(t *testing.T) {
		t.Parallel()
		req := requestWithCookie("session_id", "YWJj--deadbeef")
		_, err := signer.Verify(req)
		require.Error(t, err)
		assert.ErrorIs(t, err, httpserver.ErrMalformedValue)
	})

	t.Run("signature_empty", func(t *testing.T) {
		t.Parallel()
		req := requestWithCookie("session_id", "YWJj--")
		_, err := signer.Verify(req)
		require.Error(t, err)
		assert.ErrorIs(t, err, httpserver.ErrMalformedValue)
	})

	t.Run("payload_not_base64", func(t *testing.T) {
		t.Parallel()
		payload := "!!!"
		sig := signForTest(primaryKey(), "session_id", payload)
		req := requestWithCookie("session_id", payload+"--"+sig)
		_, err := signer.Verify(req)
		require.Error(t, err)
		assert.ErrorIs(t, err, httpserver.ErrInvalidBase64)
	})
}

func TestHMACCookieSigner_NewValidation(t *testing.T) {
	t.Parallel()

	_, err := httpserver.NewHMACCookieSigner("", primaryKey())
	require.Error(t, err)
	assert.ErrorIs(t, err, httpserver.ErrInvalidConfig)

	_, err = httpserver.NewHMACCookieSigner("session_id", nil)
	require.Error(t, err)
	assert.ErrorIs(t, err, httpserver.ErrInvalidConfig)

	_, err = httpserver.NewHMACCookieSigner("session_id", primaryKey(), nil)
	require.Error(t, err)
	assert.ErrorIs(t, err, httpserver.ErrInvalidConfig)

	_, err = httpserver.NewHMACCookieSigner("session_id", primaryKey(), []byte{})
	require.Error(t, err)
	assert.ErrorIs(t, err, httpserver.ErrInvalidConfig)
}

func TestHMACCookieSigner_Verify_NoCookie(t *testing.T) {
	t.Parallel()

	signer := mustSigner(t, "session_id", primaryKey())
	req := httptest.NewRequest(http.MethodGet, "/", nil)

	_, err := signer.Verify(req)
	require.Error(t, err)
	assert.ErrorIs(t, err, http.ErrNoCookie)
}

func TestHMACCookieSigner_Set_DefaultsAndOverrides(t *testing.T) {
	t.Parallel()

	signer := mustSigner(t, "session_id", primaryKey())

	defaultCookie := mustSetCookie(t, signer, "value", httpserver.CookieWriteOptions{})
	assert.Equal(t, "session_id", defaultCookie.Name)
	assert.Equal(t, "/", defaultCookie.Path)
	assert.True(t, defaultCookie.Secure)
	assert.True(t, defaultCookie.HttpOnly)
	assert.Equal(t, http.SameSiteNoneMode, defaultCookie.SameSite)
	assert.Empty(t, defaultCookie.Domain)
	assert.Zero(t, defaultCookie.MaxAge)
	assert.True(t, defaultCookie.Expires.IsZero())

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.AddCookie(defaultCookie)
	got, err := signer.Verify(req)
	require.NoError(t, err)
	assert.Equal(t, []byte("value"), got)

	lax := http.SameSiteLaxMode
	insecure := false
	notHTTPOnly := false
	expires := time.Unix(1_700_000_000, 0).UTC()

	overrideCookie := mustSetCookie(t, signer, "value", httpserver.CookieWriteOptions{
		Path:     "/api",
		Domain:   ".wild.io",
		MaxAge:   3600,
		Expires:  expires,
		Secure:   &insecure,
		HttpOnly: &notHTTPOnly,
		SameSite: &lax,
	})

	assert.Equal(t, "/api", overrideCookie.Path)
	assert.Equal(t, "wild.io", overrideCookie.Domain)
	assert.Equal(t, 3600, overrideCookie.MaxAge)
	assert.True(t, overrideCookie.Expires.Equal(expires))
	assert.False(t, overrideCookie.Secure)
	assert.False(t, overrideCookie.HttpOnly)
	assert.Equal(t, http.SameSiteLaxMode, overrideCookie.SameSite)
}

func mustSigner(
	t *testing.T,
	cookieName string,
	primaryKey []byte,
	fallbackKeys ...[]byte,
) httpserver.HMACCookieSigner {
	t.Helper()

	signer, err := httpserver.NewHMACCookieSigner(cookieName, primaryKey, fallbackKeys...)
	require.NoError(t, err)

	return signer
}

func mustSetCookie(
	t *testing.T,
	signer httpserver.HMACCookieSigner,
	value string,
	opts httpserver.CookieWriteOptions,
) *http.Cookie {
	t.Helper()

	rr := httptest.NewRecorder()
	err := signer.Set(rr, value, opts)
	require.NoError(t, err)

	resp := rr.Result()
	t.Cleanup(func() {
		_ = resp.Body.Close()
	})

	cookies := resp.Cookies()
	require.Len(t, cookies, 1)

	return cookies[0]
}

func requestWithCookie(name, value string) *http.Request {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.AddCookie(&http.Cookie{Name: name, Value: value})

	return req
}

func primaryKey() []byte {
	return []byte("primary-key-32-bytes-abcdefghijkl")
}

func mutateHexDigit(sig string) string {
	if len(sig) == 0 {
		return sig
	}

	last := sig[len(sig)-1]
	switch last {
	case '0':
		return sig[:len(sig)-1] + "1"
	default:
		return sig[:len(sig)-1] + "0"
	}
}

func signForTest(key []byte, name, payload string) string {
	mac := hmac.New(sha256.New, key)
	_, _ = mac.Write([]byte(name))
	_, _ = mac.Write([]byte{0})
	_, _ = mac.Write([]byte(payload))

	return hex.EncodeToString(mac.Sum(nil))
}
