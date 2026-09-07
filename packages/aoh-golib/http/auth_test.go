package aohhttp_test

import (
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	aohhttp "github.com/mssfoobar/ops-hub/packages/aoh-golib/http"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"
)

// unsignedJWT builds a structurally valid, unverified token — enough for the
// ParseUnverified the middleware does locally before calling userinfo.
func unsignedJWT(subject string) string {
	enc := func(v any) string {
		raw, _ := json.Marshal(v)
		return base64.RawURLEncoding.EncodeToString(raw)
	}
	return enc(map[string]string{"alg": "none", "typ": "JWT"}) + "." +
		enc(map[string]string{"sub": subject}) + ".signature"
}

// TestBearerAuth covers the middleware's partitions in one function on purpose:
// aohhttp's OpenID cache is a process-wide singleton with a 24h TTL, so the
// first discovery in the package wins. One issuer, one discovery, and the
// userinfo verdict is switched per subtest.
func TestBearerAuth(t *testing.T) {
	userinfoStatus := http.StatusUnauthorized

	mux := http.NewServeMux()
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)
	mux.HandleFunc("/.well-known/openid-configuration", func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]string{
			"issuer":            server.URL,
			"userinfo_endpoint": server.URL + "/userinfo",
		})
	})
	mux.HandleFunc("/userinfo", func(w http.ResponseWriter, _ *http.Request) {
		if userinfoStatus != http.StatusOK {
			w.WriteHeader(userinfoStatus)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"sub":           "u1",
			"active_tenant": map[string]any{"tenant_id": "t1"},
		})
	})

	authed := func(t *testing.T, token string, next http.Handler) (*http.Response, *observer.ObservedLogs) {
		t.Helper()
		logger, logs := observed()
		req := httptest.NewRequest(http.MethodGet, "/widgets", nil)
		if token != "" {
			req.Header.Set("Authorization", "Bearer "+token)
		}
		res := serve(t, aohhttp.BearerAuth(server.URL, logger)(next), req)
		return res, logs
	}

	// Closes the empty-body hole: the middleware used to answer a rejected
	// token with a bare w.WriteHeader(401).
	t.Run("rejected token renders a conformant 401", func(t *testing.T) {
		userinfoStatus = http.StatusUnauthorized

		res, logs := authed(t, unsignedJWT("u1"), unreachableHandler(t))

		assert.Equal(t, http.StatusUnauthorized, res.StatusCode)
		assert.Equal(t, "application/json", res.Header.Get("Content-Type"))
		keys, _ := jsonBody(t, res)
		assert.Equal(t, "UNAUTHENTICATED", strValue(t, keys, "errorCode"))
		assert.NotEmpty(t, strValue(t, keys, "errorMessage"))
		assert.NotEmpty(t, strValue(t, keys, "timestamp"))

		entry := onlyRecord(t, logs)
		assert.Equal(t, zapcore.WarnLevel, entry.Level,
			"a rejected credential is not an alert, and it is reported once")
	})

	t.Run("malformed token renders a conformant 401", func(t *testing.T) {
		userinfoStatus = http.StatusOK

		res, _ := authed(t, "not-a-jwt", unreachableHandler(t))

		assert.Equal(t, http.StatusUnauthorized, res.StatusCode)
		keys, _ := jsonBody(t, res)
		assert.Equal(t, "UNAUTHENTICATED", strValue(t, keys, "errorCode"))
	})

	t.Run("missing header renders a conformant 401", func(t *testing.T) {
		userinfoStatus = http.StatusOK

		res, _ := authed(t, "", unreachableHandler(t))

		assert.Equal(t, http.StatusUnauthorized, res.StatusCode)
		keys, _ := jsonBody(t, res)
		assert.Equal(t, "UNAUTHENTICATED", strValue(t, keys, "errorCode"))
	})

	// The credential must survive neither in the body nor in any record field.
	t.Run("neither body nor record carries the token", func(t *testing.T) {
		userinfoStatus = http.StatusUnauthorized
		token := unsignedJWT("u1")

		res, logs := authed(t, token, unreachableHandler(t))

		_, body := jsonBody(t, res)
		assert.NotContains(t, body, token)
		for key, value := range onlyRecord(t, logs).ContextMap() {
			if s, ok := value.(string); ok {
				assert.NotContains(t, s, token, "field %q leaked the token", key)
			}
		}
	})

	// The happy path the renderer change must not touch.
	t.Run("valid token passes through", func(t *testing.T) {
		userinfoStatus = http.StatusOK
		reached := false

		res, _ := authed(t, unsignedJWT("u1"), http.HandlerFunc(
			func(w http.ResponseWriter, _ *http.Request) {
				reached = true
				w.WriteHeader(http.StatusNoContent)
			}))

		assert.True(t, reached, "a valid token must reach the handler")
		assert.Equal(t, http.StatusNoContent, res.StatusCode)
	})
}

func unreachableHandler(t *testing.T) http.Handler {
	t.Helper()
	return http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		t.Error("the handler must not be reached when auth fails")
	})
}
