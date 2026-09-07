// Package activities holds this worker's Temporal activities. Every exported
// method on Activities with the signature func(context.Context, ...) (T, error)
// is registered as a Temporal activity, addressable by its method name — exactly
// the string a DSL `Activity` step puts in its `Type` (and what the WFE
// `service_activity` registry lists for the designer).
//
// Replace HelloWorld with your own activities.
package activities

import (
	"context"
	"fmt"
	"time"

	aohlog "github.com/mssfoobar/ops-hub/packages/aoh-golib/logger"
	"go.temporal.io/sdk/activity"
	"go.uber.org/zap"
)

// Activities is the receiver the worker registers (see cmd/worker/main.go).
type Activities struct{}

// HelloWorld is a starter ACTIVITY: given a name, it returns a greeting. It shows
// the shape every activity follows — a context, typed parameters, and a
// (result, error) return. Delete it once you have your own.
func (a *Activities) HelloWorld(ctx context.Context, name string) (string, error) {
	aohlog.Info("HelloWorld invoked",
		zap.String("activity", activity.GetInfo(ctx).ActivityType.Name),
		zap.String("name", name))
	return fmt.Sprintf("Hello, %s!", name), nil
}

// Timer is a starter EVENT handler that waits `seconds`, then completes. It is
// the reference for the two patterns EVERY long-running activity needs:
//
//   - Heartbeat. It calls activity.RecordHeartbeat each tick. The engine runs
//     events with a 1-minute HeartbeatTimeout (see setEventOption), so an activity
//     that stops heartbeating is failed — and heartbeating is also what DELIVERS
//     cancellation to ctx.
//   - Cancellable. It selects on ctx.Done() and returns promptly when the workflow
//     cancels it — e.g. an interrupting boundary timer, or a racing path that
//     finishes first. The engine runs events WaitForCancellation, so it waits for
//     this acknowledgement. Without it, an "interrupting" event couldn't interrupt.
//
// Events are worker methods *just like* activities — registered the same way and
// dispatched by method name; the only difference is the catalog (`service_event`,
// with a timer icon) and the BPMN node the designer draws. Delete it once you have
// your own.
func (a *Activities) Timer(ctx context.Context, seconds float64) error {
	name := activity.GetInfo(ctx).ActivityType.Name
	total := time.Duration(seconds * float64(time.Second))
	const tick = time.Second
	for elapsed := time.Duration(0); elapsed < total; {
		activity.RecordHeartbeat(ctx, elapsed.Seconds())
		wait := tick
		if remaining := total - elapsed; remaining < wait {
			wait = remaining
		}
		select {
		case <-ctx.Done():
			aohlog.Info("Timer interrupted",
				zap.String("event", name), zap.Float64("elapsed_s", elapsed.Seconds()))
			return ctx.Err()
		case <-time.After(wait):
			elapsed += wait
		}
	}
	aohlog.Info("Timer fired", zap.String("event", name), zap.Float64("seconds", seconds))
	return nil
}
