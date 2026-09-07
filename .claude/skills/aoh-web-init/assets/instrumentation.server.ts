import { startObservability } from '@mssfoobar/observability/sveltekit';

// OpenTelemetry bootstrap. SvelteKit imports this file before any
// other server module, so auto-instrumentation can patch http/pino/etc. first.
// A no-op until OTEL_EXPORTER_OTLP_ENDPOINT is set (point it at the per-spoke
// gateway Collector, e.g. http://otel-collector:4317) — safe in every env.
// Logs stay on pino with trace_id/span_id injected; see hooks.server.ts for the
// route-pattern span name + team_id baggage hook.
startObservability({ serviceName: 'web-{APP_NAME}' });
