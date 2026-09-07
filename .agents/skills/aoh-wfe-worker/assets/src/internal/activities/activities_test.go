package activities

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.temporal.io/sdk/testsuite"
)

func TestHelloWorld(t *testing.T) {
	a := &Activities{}
	env := (&testsuite.WorkflowTestSuite{}).NewTestActivityEnvironment()
	env.RegisterActivity(a)

	val, err := env.ExecuteActivity(a.HelloWorld, "WFE")
	require.NoError(t, err)
	var result string
	require.NoError(t, val.Get(&result))
	assert.Equal(t, "Hello, WFE!", result)
}

func TestTimer(t *testing.T) {
	a := &Activities{}
	env := (&testsuite.WorkflowTestSuite{}).NewTestActivityEnvironment()
	env.RegisterActivity(a)

	// A short positive duration exercises the wait loop + heartbeat and completes
	// with nil (seconds=0 would skip the loop entirely). Cancellation — returning
	// ctx.Err() on ctx.Done() — is delivered by the engine at runtime; see the
	// Timer doc comment.
	_, err := env.ExecuteActivity(a.Timer, 0.05)
	require.NoError(t, err)
}
