package aohhttp_test

import (
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	aohhttp "github.com/mssfoobar/ops-hub/packages/aoh-golib/http"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// isValidJWT compared the userinfo `sub` through an unchecked type assertion, so
// ANY bearer token without a sub claim panicked — on the authentication path of
// every Go service, on attacker-controlled input. aohhttp.Recoverer turned that
// into a 500, so a malformed credential was reported as a server fault rather
// than as a rejected one, and each one cost a panic and a recover.
//
// Falsified: restoring `parsed["sub"].(string)` makes this fail on the panic.
func TestValidateJWTWithRetry_TokenWithoutSubIsRejectedNotPanicking(t *testing.T) {
	mux := http.NewServeMux()
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)
	mux.HandleFunc("/.well-known/openid-configuration", func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]string{
			"issuer": server.URL, "userinfo_endpoint": server.URL + "/userinfo",
		})
	})
	mux.HandleFunc("/userinfo", func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{"sub": "someone"})
	})

	header := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"RS256","typ":"JWT"}`))
	payload := base64.RawURLEncoding.EncodeToString([]byte(`{"exp":9999999999}`)) // no sub
	token := header + "." + payload + ".signature"

	require.NotPanics(t, func() {
		err := aohhttp.ValidateJWTWithRetry(server.URL, token)
		assert.Error(t, err, "a token with no sub must be rejected, not accepted")
	})
}
