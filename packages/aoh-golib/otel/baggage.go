package aohotel

import (
	"context"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/baggage"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
)

// TeamIDKey is the baggage / attribute key for the team (group) a request
// belongs to. In AOH a "team"/"tenant" is an access-grouping, NOT customer
// isolation, so it's fine to carry on spans and logs for filtering. It MUST NOT
// be used as a metric label/attribute (cardinality), and secrets/PII MUST NOT
// be put in baggage (it propagates to downstream services and the wire).
const TeamIDKey = "team_id"

// ContextWithTeamID returns ctx with team_id added to W3C baggage so downstream
// services — and this service's spans/logs — can stamp it. No-op for an empty
// id or an id that isn't a valid baggage value.
func ContextWithTeamID(ctx context.Context, teamID string) context.Context {
	if teamID == "" {
		return ctx
	}
	member, err := baggage.NewMember(TeamIDKey, teamID)
	if err != nil {
		return ctx
	}
	bag, err := baggage.FromContext(ctx).SetMember(member)
	if err != nil {
		return ctx
	}
	return baggage.ContextWithBaggage(ctx, bag)
}

// TeamID returns the team_id carried in ctx's baggage, or "" if absent.
func TeamID(ctx context.Context) string {
	return baggage.FromContext(ctx).Member(TeamIDKey).Value()
}

// TeamIDField returns a zap field for the team_id in ctx, or zap.Skip() when
// absent — safe to splat into log calls alongside TraceContextFields.
func TeamIDField(ctx context.Context) zap.Field {
	if v := TeamID(ctx); v != "" {
		return zap.String(TeamIDKey, v)
	}
	return zap.Skip()
}

// SetTeamIDOnSpan stamps the recording span in ctx with the team_id attribute
// (read from baggage). No-op when there's no team_id or no recording span.
// Use on spans/logs only — never add team_id to metric instruments.
func SetTeamIDOnSpan(ctx context.Context) {
	v := TeamID(ctx)
	if v == "" {
		return
	}
	if span := trace.SpanFromContext(ctx); span.IsRecording() {
		span.SetAttributes(attribute.String(TeamIDKey, v))
	}
}
