package aohhttp

import (
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/mssfoobar/ops-hub/packages/aoh-golib/aoherr"
	aohlog "github.com/mssfoobar/ops-hub/packages/aoh-golib/logger"
	"go.uber.org/zap"
)

// accessMessage is the constant message of the access record. Its identity is in
// its fields, so it stays queryable.
const accessMessage = "response"

// BearerAuth checks if the jwt is valid by calling openID provider endpoint.
//
// A rejected token is rendered as the conformant AOH 401 (previously a bare
// w.WriteHeader, which handed clients a zero-length body). The renderer is also
// the single logging point, so the record lands at WARN — a rejected credential
// is a client fault, not an alert — and carries no part of the token.
//
// example issuerUrl - http://iams-keycloak:8080/realms/AOH
func BearerAuth(issuerUrl string, logger *zap.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			token := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")

			if err := ValidateJWTWithRetry(issuerUrl, token); err != nil {
				aoherr.RenderWithLogger(logger, w, r, aoherr.Wrap(
					aoherr.ClassAuthentication, aoherr.CodeUnauthenticated,
					"the bearer token is missing, malformed, expired or rejected by the issuer",
					err))
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// RequestLogger logs the request
func RequestLogger(logger *zap.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			aohlog.WithCtx(r.Context(), logger).Debug(
				"request",
				zap.String("method", r.Method),
				zap.String("path", r.RequestURI),
			)
			next.ServeHTTP(w, r)
		})
	}
}

// ResponseLogger emits one bounded access record per response.
//
// It is deliberately *not* a failure report: it carries status, method, route,
// duration and response size, and never the body, the errorCode or the cause —
// so it cannot double-report a failure aoherr already logged (the one-record
// rule). Its level follows the status (4xx WARN, 5xx ERROR, otherwise DEBUG);
// it used to log all of 400-599 at ERROR.
//
// The response streams straight through to the client. The previous
// implementation buffered every response through httptest.NewRecorder in order
// to log its body, which put whole payloads in memory on the hot path.
func ResponseLogger(logger *zap.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)
			start := time.Now()

			next.ServeHTTP(ww, r)

			logAccess(logger, r, ww.Status(), ww.BytesWritten(), time.Since(start))
		})
	}
}

func logAccess(logger *zap.Logger, r *http.Request, status, size int, elapsed time.Duration) {
	if status == 0 {
		// The handler returned without calling WriteHeader; net/http implies 200.
		status = http.StatusOK
	}

	fields := []zap.Field{
		zap.Int("status", status),
		zap.String("method", r.Method),
		zap.Duration("duration", elapsed),
		zap.Int("bytes", size),
	}
	if route := chiRoutePattern(r); route != "" {
		fields = append(fields, zap.String("route", route))
	}

	l := aohlog.WithCtx(r.Context(), logger)
	switch {
	case status >= http.StatusInternalServerError:
		l.Error(accessMessage, fields...)
	case status >= http.StatusBadRequest:
		l.Warn(accessMessage, fields...)
	default:
		l.Debug(accessMessage, fields...)
	}
}
