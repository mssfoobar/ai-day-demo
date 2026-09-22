//go:build integration

// Repo tests that must run against a real PostgreSQL, from DISPATCH_SVC_TEST_DSN and
// skipped when it is unset.
//
//	DISPATCH_SVC_TEST_DSN='host=localhost port=5432 user=dispatch password=dispatch dbname=dispatch sslmode=disable' \
//	  go test -tags=integration ./internal/repo/... -count=1 -race
//
// Or, from the repo root: `pnpm --filter @mssfoobar/dispatch-svc test:integration`.
//
// These cannot be faked. The outbox's whole promise is that the row and the unit share a
// transaction, and the seed's guard is `INSERT … ON CONFLICT DO NOTHING RETURNING` —
// both are properties of PostgreSQL, not of Go. A fake that returned what it was told
// would prove nothing about either.
package repo_test

import (
	"context"
	"fmt"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/mssfoobar/fleet-dispatch-console/apps/dispatch-svc/internal/db"
	"github.com/mssfoobar/fleet-dispatch-console/apps/dispatch-svc/internal/domain"
	"github.com/mssfoobar/fleet-dispatch-console/apps/dispatch-svc/internal/projection"
	"github.com/mssfoobar/fleet-dispatch-console/apps/dispatch-svc/internal/repo"
)

const testSchema = "dispatch_it"

func testDB(t *testing.T) *sqlx.DB {
	t.Helper()
	dsn := os.Getenv("DISPATCH_SVC_TEST_DSN")
	if dsn == "" {
		t.Skip("DISPATCH_SVC_TEST_DSN is unset")
	}

	pool, err := db.Open(context.Background(), dsn)
	require.NoError(t, err)
	t.Cleanup(func() { _ = pool.Close() })

	// Self-migrating into its own schema, so the tests never touch the schema a running
	// dispatch-svc is using.
	require.NoError(t, db.Migrate(context.Background(), pool, testSchema))
	return pool
}

// A unique tenant per test keeps runs isolated without truncating tables — and it is
// also what makes the seed reachable more than once, since the marker is per tenant.
func testCaller(t *testing.T) repo.Caller {
	t.Helper()
	return repo.Caller{
		TenantID: fmt.Sprintf("test-tenant-%s-%d", t.Name(), time.Now().UnixNano()),
		Subject:  "test-subject",
	}
}

func newInput(code string, pos *domain.PositionInput) domain.UnitInput {
	return domain.UnitInput{
		UnitCode: code, CallSign: "Test " + code, Status: domain.StatusAvailable,
		UnitType: "Ambulance", Station: "Station", Sector: "Sector",
		RadioChannel: "TAC-1", Shift: "Day", Capabilities: []string{},
		Position: pos,
	}
}

func countOutbox(t *testing.T, pool *sqlx.DB, caller repo.Caller, intent projection.Intent) int {
	t.Helper()
	var n int
	require.NoError(t, pool.Get(&n, fmt.Sprintf(
		`SELECT count(*) FROM %s.gis_outbox WHERE tenant_id = $1 AND intent = $2`, testSchema),
		caller.TenantID, string(intent)))
	return n
}

// --------------------------------------------------------------------------- //
// The outbox row and the unit share a transaction
// --------------------------------------------------------------------------- //

func TestCreate_EnqueuesTheProjectionInTheSameTransaction(t *testing.T) {
	pool := testDB(t)
	r := repo.NewUnitRepo(pool, testSchema)
	caller := testCaller(t)

	_, rows, err := r.Create(context.Background(), caller,
		newInput("FU-IT1", &domain.PositionInput{Lon: 103.85, Lat: 1.29}))
	require.NoError(t, err)

	require.Len(t, rows, 1)
	assert.Equal(t, projection.IntentUpsert, rows[0].Intent)
	assert.Equal(t, 1, countOutbox(t, pool, caller, projection.IntentUpsert))
}

func TestCreate_AnUnpositionedUnitEnqueuesADeleteNotAnUpsert(t *testing.T) {
	pool := testDB(t)
	r := repo.NewUnitRepo(pool, testSchema)
	caller := testCaller(t)

	_, rows, err := r.Create(context.Background(), caller, newInput("FU-IT2", nil))
	require.NoError(t, err)

	require.Len(t, rows, 1)
	assert.Equal(t, projection.IntentDelete, rows[0].Intent,
		"a unit with no coordinates has nothing to project")
	assert.Zero(t, countOutbox(t, pool, caller, projection.IntentUpsert))
	assert.Equal(t, 1, countOutbox(t, pool, caller, projection.IntentDelete))
}

// The transaction is the whole point: a unit write that rolls back must leave no pending
// projection behind, or the mirror would describe a unit that was never stored.
func TestCreate_ARolledBackWriteLeavesNoOutboxRow(t *testing.T) {
	pool := testDB(t)
	r := repo.NewUnitRepo(pool, testSchema)
	caller := testCaller(t)

	_, _, err := r.Create(context.Background(), caller,
		newInput("FU-IT3", &domain.PositionInput{Lon: 103.85, Lat: 1.29}))
	require.NoError(t, err)
	require.Equal(t, 1, countOutbox(t, pool, caller, projection.IntentUpsert))

	// The unique constraint rejects the second insert, which rolls the whole
	// transaction back — enqueue included.
	_, _, err = r.Create(context.Background(), caller,
		newInput("FU-IT3", &domain.PositionInput{Lon: 1, Lat: 1}))
	require.ErrorIs(t, err, repo.ErrConflict)

	assert.Equal(t, 1, countOutbox(t, pool, caller, projection.IntentUpsert),
		"the failed write must not have enqueued a second projection")
}

func TestUpdate_ClearingThePositionEnqueuesADelete(t *testing.T) {
	pool := testDB(t)
	r := repo.NewUnitRepo(pool, testSchema)
	caller := testCaller(t)
	ctx := context.Background()

	unit, _, err := r.Create(ctx, caller, newInput("FU-IT4", &domain.PositionInput{Lon: 103.85, Lat: 1.29}))
	require.NoError(t, err)

	cleared, rows, err := r.Update(ctx, caller, "FU-IT4", unit.OccLock, newInput("FU-IT4", nil))
	require.NoError(t, err)

	assert.Nil(t, cleared.Position, "a replace with no position clears it")
	require.Len(t, rows, 1)
	assert.Equal(t, projection.IntentDelete, rows[0].Intent,
		"otherwise a stale marker stays on the map at the unit's last known location")
	assert.Equal(t, unit.OccLock+1, cleared.OccLock, "a position change is a replace like any other")
}

func TestDelete_EnqueuesTheEntityRemoval(t *testing.T) {
	pool := testDB(t)
	r := repo.NewUnitRepo(pool, testSchema)
	caller := testCaller(t)
	ctx := context.Background()

	unit, _, err := r.Create(ctx, caller, newInput("FU-IT5", &domain.PositionInput{Lon: 103.85, Lat: 1.29}))
	require.NoError(t, err)

	rows, err := r.Delete(ctx, caller, "FU-IT5", unit.OccLock)
	require.NoError(t, err)

	require.Len(t, rows, 1)
	assert.Equal(t, projection.IntentDelete, rows[0].Intent)
	assert.Equal(t, "FU-IT5", rows[0].UnitCode)
}

// --------------------------------------------------------------------------- //
// Position and last contact move independently (design.md D11)
// --------------------------------------------------------------------------- //

func TestUpdate_AnEditThatRepeatsThePositionDoesNotMoveTheFixTime(t *testing.T) {
	pool := testDB(t)
	r := repo.NewUnitRepo(pool, testSchema)
	caller := testCaller(t)
	ctx := context.Background()

	fix := time.Now().UTC().Add(-90 * time.Minute).Truncate(time.Millisecond)
	unit, _, err := r.Create(ctx, caller,
		newInput("FU-IT6", &domain.PositionInput{Lon: 103.85, Lat: 1.29, At: &fix}))
	require.NoError(t, err)
	require.NotNil(t, unit.Position)
	firstContact := unit.LastContact

	// An operator editing the radio channel is a contact, not a position report. The
	// console round-trips the existing fix, so `at` comes back unchanged.
	edited := newInput("FU-IT6", &domain.PositionInput{Lon: 103.85, Lat: 1.29, At: &fix})
	edited.RadioChannel = "TAC-9"

	updated, _, err := r.Update(ctx, caller, "FU-IT6", unit.OccLock, edited)
	require.NoError(t, err)

	require.NotNil(t, updated.Position)
	assert.WithinDuration(t, fix, updated.Position.At, time.Millisecond,
		"the fix time is when the location was reported, not when the row was touched")
	assert.True(t, updated.LastContact.After(firstContact),
		"last contact still advances on every write")
}

// --------------------------------------------------------------------------- //
// The seed guard
// --------------------------------------------------------------------------- //

func TestSeedTenant_TwoConcurrentDispatchersSeedExactlyOnce(t *testing.T) {
	pool := testDB(t)
	r := repo.NewUnitRepo(pool, testSchema)
	caller := testCaller(t)

	roster := []domain.SeedUnit{
		{UnitInput: newInput("FU-S1", &domain.PositionInput{Lon: 103.85, Lat: 1.29}),
			Crew: []domain.Crew{{Name: "A", Role: "Driver"}, {Name: "B", Role: "EMT"}}},
		{UnitInput: newInput("FU-S2", nil),
			Crew: []domain.Crew{{Name: "C", Role: "Officer"}}},
	}

	// Both racers use the same tenant and differ only in who they are, which is exactly
	// the two-simultaneous-sign-ins case.
	const racers = 2
	var (
		wg      sync.WaitGroup
		mu      sync.Mutex
		wins    int
		rowsGot int
		errs    []error
	)
	start := make(chan struct{})
	for i := 0; i < racers; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			c := caller
			c.Subject = fmt.Sprintf("racer-%d", i)
			<-start
			seeded, rows, err := r.SeedTenant(context.Background(), c, roster)
			mu.Lock()
			defer mu.Unlock()
			if err != nil {
				errs = append(errs, err)
				return
			}
			if seeded {
				wins++
				rowsGot += len(rows)
			}
		}(i)
	}
	close(start)
	wg.Wait()

	// A racer that loses the marker may also lose a serialisation race on the marker row
	// itself; what must never happen is two winners.
	assert.LessOrEqual(t, len(errs), racers-1, "at most the losers may error: %v", errs)
	require.Equal(t, 1, wins, "exactly one racer may seed the tenant")
	assert.Equal(t, len(roster), rowsGot, "exactly one set of projections is enqueued")

	var units, crew, markers, outbox, maxOcc int
	require.NoError(t, pool.Get(&units, fmt.Sprintf(
		`SELECT count(*) FROM %s.unit WHERE tenant_id = $1`, testSchema), caller.TenantID))
	require.NoError(t, pool.Get(&crew, fmt.Sprintf(
		`SELECT count(*) FROM %s.unit_crew WHERE tenant_id = $1`, testSchema), caller.TenantID))
	require.NoError(t, pool.Get(&markers, fmt.Sprintf(
		`SELECT count(*) FROM %s.tenant_seed WHERE tenant_id = $1`, testSchema), caller.TenantID))
	require.NoError(t, pool.Get(&outbox, fmt.Sprintf(
		`SELECT count(*) FROM %s.gis_outbox WHERE tenant_id = $1`, testSchema), caller.TenantID))
	require.NoError(t, pool.Get(&maxOcc, fmt.Sprintf(
		`SELECT coalesce(max(occ_lock), 0) FROM %s.unit WHERE tenant_id = $1`, testSchema), caller.TenantID))

	assert.Equal(t, 2, units, "no duplicate units")
	assert.Equal(t, 3, crew, "no duplicate crew")
	assert.Equal(t, 1, markers)
	assert.Equal(t, 2, outbox, "exactly one set of projections")
	assert.Zero(t, maxOcc, "no seeded unit was bumped by a second write")
}

func TestSeedTenant_IsNotDecidedByWhetherTheTenantHasUnits(t *testing.T) {
	// Deleting units is a legitimate operator action and must not resurrect the roster:
	// the marker survives an emptied tenant.
	pool := testDB(t)
	r := repo.NewUnitRepo(pool, testSchema)
	caller := testCaller(t)
	ctx := context.Background()

	roster := []domain.SeedUnit{{UnitInput: newInput("FU-S3", nil)}}

	seeded, _, err := r.SeedTenant(ctx, caller, roster)
	require.NoError(t, err)
	require.True(t, seeded)

	unit, err := r.Get(ctx, caller, "FU-S3")
	require.NoError(t, err)
	_, err = r.Delete(ctx, caller, "FU-S3", unit.OccLock)
	require.NoError(t, err)

	isSeeded, err := r.IsSeeded(ctx, caller.TenantID)
	require.NoError(t, err)
	assert.True(t, isSeeded, "the marker outlives the units it created")

	seeded, _, err = r.SeedTenant(ctx, caller, roster)
	require.NoError(t, err)
	assert.False(t, seeded, "an emptied tenant is not reseeded")

	units, err := r.List(ctx, caller)
	require.NoError(t, err)
	assert.Empty(t, units, "the roster stays empty")
}

// --------------------------------------------------------------------------- //
// Tenant scoping, against the real predicate
// --------------------------------------------------------------------------- //

func TestTenantScoping_AnotherTenantsUnitIsInvisibleAndUnwritable(t *testing.T) {
	pool := testDB(t)
	r := repo.NewUnitRepo(pool, testSchema)
	ctx := context.Background()

	mine := testCaller(t)
	theirs := mine
	theirs.TenantID += "-other"

	_, _, err := r.Create(ctx, theirs, newInput("FU-X1", nil))
	require.NoError(t, err)

	units, err := r.List(ctx, mine)
	require.NoError(t, err)
	assert.Empty(t, units, "another tenant's unit is absent from the list")

	_, err = r.Get(ctx, mine, "FU-X1")
	assert.ErrorIs(t, err, repo.ErrNotFound)

	_, _, err = r.Update(ctx, mine, "FU-X1", 0, newInput("FU-X1", nil))
	assert.ErrorIs(t, err, repo.ErrNotFound, "not ErrStale: the row is not ours to be stale about")

	_, err = r.Delete(ctx, mine, "FU-X1", 0)
	assert.ErrorIs(t, err, repo.ErrNotFound)

	// The same code may exist in two tenants — unit_code is unique per tenant, not
	// globally, which is safe because GIS is itself tenant-partitioned.
	_, _, err = r.Create(ctx, mine, newInput("FU-X1", nil))
	assert.NoError(t, err)

	theirUnit, err := r.Get(ctx, theirs, "FU-X1")
	require.NoError(t, err)
	assert.Zero(t, theirUnit.OccLock, "their row was never touched")
}
