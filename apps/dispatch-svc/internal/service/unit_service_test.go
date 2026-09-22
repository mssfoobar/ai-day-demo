package service_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/mssfoobar/ops-hub/packages/aoh-golib/aoherr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/mssfoobar/fleet-dispatch-console/apps/dispatch-svc/internal/auth"
	"github.com/mssfoobar/fleet-dispatch-console/apps/dispatch-svc/internal/domain"
	"github.com/mssfoobar/fleet-dispatch-console/apps/dispatch-svc/internal/projection"
	"github.com/mssfoobar/fleet-dispatch-console/apps/dispatch-svc/internal/repo"
	"github.com/mssfoobar/fleet-dispatch-console/apps/dispatch-svc/internal/service"
)

const (
	tenantA    = "tenant-a"
	tenantB    = "tenant-b"
	subjectAda = "ada"
)

// --------------------------------------------------------------------------- //
// Doubles
// --------------------------------------------------------------------------- //

// fakeStore records what the service asked of the persistence layer.
//
// Hand-written rather than generated: the repo surface is small, the service already
// declares it at the consumer (service.UnitStore), and this is the first test in the
// repository — so it establishes the pattern rather than importing a mock framework
// the service has never needed.
type fakeStore struct {
	// Recorded calls.
	callers   []repo.Caller
	created   []domain.UnitInput
	updated   []domain.UnitInput
	deleted   []string
	seedCalls int

	// Programmed responses.
	unit       domain.Unit
	list       []domain.Unit
	rows       []projection.Row
	seeded     bool
	seedDidRun bool
	err        error
	seedErr    error
	isSeedErr  error
}

func (f *fakeStore) record(c repo.Caller) { f.callers = append(f.callers, c) }

func (f *fakeStore) List(_ context.Context, c repo.Caller) ([]domain.Unit, error) {
	f.record(c)
	return f.list, f.err
}

func (f *fakeStore) Get(_ context.Context, c repo.Caller, _ string) (domain.Unit, error) {
	f.record(c)
	return f.unit, f.err
}

func (f *fakeStore) Create(_ context.Context, c repo.Caller, in domain.UnitInput) (domain.Unit, []projection.Row, error) {
	f.record(c)
	f.created = append(f.created, in)
	return f.unit, f.rows, f.err
}

func (f *fakeStore) Update(_ context.Context, c repo.Caller, _ string, _ int, in domain.UnitInput) (domain.Unit, []projection.Row, error) {
	f.record(c)
	f.updated = append(f.updated, in)
	return f.unit, f.rows, f.err
}

func (f *fakeStore) Delete(_ context.Context, c repo.Caller, unitCode string, _ int) ([]projection.Row, error) {
	f.record(c)
	f.deleted = append(f.deleted, unitCode)
	return f.rows, f.err
}

func (f *fakeStore) IsSeeded(_ context.Context, _ string) (bool, error) {
	return f.seeded, f.isSeedErr
}

func (f *fakeStore) SeedTenant(_ context.Context, c repo.Caller, roster []domain.SeedUnit) (bool, []projection.Row, error) {
	f.seedCalls++
	f.record(c)
	if f.seedErr != nil {
		return false, nil, f.seedErr
	}
	rows := make([]projection.Row, 0, len(roster))
	for _, seed := range roster {
		rows = append(rows, projection.Row{UnitCode: seed.UnitCode})
	}
	f.seedDidRun = true
	return true, rows, nil
}

// fakeProjector captures what was handed to the projection worker. Buffered so the
// service's goroutine-free Deliver call never blocks, and so a test can assert on order.
type delivery struct {
	rows   []projection.Row
	token  string
	expiry time.Time
}

type fakeProjector struct{ delivered chan delivery }

func newFakeProjector() *fakeProjector {
	return &fakeProjector{delivered: make(chan delivery, 16)}
}

func (f *fakeProjector) Deliver(rows []projection.Row, token string, expiry time.Time) {
	f.delivered <- delivery{rows: rows, token: token, expiry: expiry}
}

func (f *fakeProjector) next(t *testing.T) delivery {
	t.Helper()
	select {
	case d := <-f.delivered:
		return d
	case <-time.After(time.Second):
		t.Fatal("no projection was handed to the worker within 1s")
		return delivery{}
	}
}

func (f *fakeProjector) assertNothingElse(t *testing.T) {
	t.Helper()
	select {
	case d := <-f.delivered:
		t.Fatalf("unexpected extra projection: %+v", d)
	default:
	}
}

// --------------------------------------------------------------------------- //
// Helpers
// --------------------------------------------------------------------------- //

func ctxAs(tenant, subject string, roles ...string) context.Context {
	return auth.WithIdentity(context.Background(), auth.Identity{
		Subject:     subject,
		TenantID:    tenant,
		Roles:       roles,
		BearerToken: "token-for-" + subject,
		ExpiresAt:   time.Now().Add(5 * time.Minute),
	})
}

func dispatcherCtx() context.Context { return ctxAs(tenantA, subjectAda, service.RoleDispatcher) }
func viewerCtx() context.Context     { return ctxAs(tenantA, "bob", service.RoleViewer) }

// seededStore is a store whose tenant has already been seeded, so a test can exercise
// something other than the seed.
func seededStore() *fakeStore { return &fakeStore{seeded: true} }

func newService(store service.UnitStore, p service.Projector) *service.UnitService {
	return service.NewUnitService(store, p)
}

func validInput() domain.UnitInput {
	return domain.UnitInput{
		UnitCode: "FU-700", CallSign: "Test-1", Status: domain.StatusAvailable,
		UnitType: "Ambulance", Station: "Station 1", Sector: "Sector 1",
		RadioChannel: "TAC-1", Shift: "Day", Capabilities: []string{"ALS"},
	}
}

func assertAOH(t *testing.T, err error, class aoherr.Class, code aoherr.Code) *aoherr.Error {
	t.Helper()
	var aohErr *aoherr.Error
	require.ErrorAs(t, err, &aohErr)
	assert.Equal(t, class, aohErr.Class())
	assert.Equal(t, code, aohErr.Code())
	return aohErr
}

// --------------------------------------------------------------------------- //
// Tenant scoping and identity
// --------------------------------------------------------------------------- //

func TestList_ScopesToTheCallersTenant(t *testing.T) {
	store := seededStore()
	svc := newService(store, newFakeProjector())

	_, err := svc.List(ctxAs(tenantB, "carol", service.RoleViewer))
	require.NoError(t, err)

	require.Len(t, store.callers, 1)
	assert.Equal(t, tenantB, store.callers[0].TenantID)
	assert.Equal(t, "carol", store.callers[0].Subject)
}

func TestCreate_StampsIdentityFromTheTokenNotTheBody(t *testing.T) {
	store := seededStore()
	svc := newService(store, newFakeProjector())

	in := validInput()
	in.IgnoredTenantID = "attacker-tenant"
	in.IgnoredCreatedBy = "attacker"
	in.IgnoredUpdatedBy = "attacker"

	_, err := svc.Create(dispatcherCtx(), in)
	require.NoError(t, err)

	require.Len(t, store.callers, 1)
	assert.Equal(t, tenantA, store.callers[0].TenantID, "tenant comes from the token")
	assert.Equal(t, subjectAda, store.callers[0].Subject, "author comes from the token")
}

func TestGet_AnotherTenantsUnitIsNotFound(t *testing.T) {
	// The repo's tenant predicate turns a cross-tenant read into its not-found sentinel;
	// this pins the service's translation of that, so the two stay one-way coupled.
	store := seededStore()
	store.err = repo.ErrNotFound
	svc := newService(store, newFakeProjector())

	_, err := svc.Get(dispatcherCtx(), "FU-999")
	assertAOH(t, err, aoherr.ClassNotFound, service.CodeUnitNotFound)
}

func TestList_WithNoIdentityIsASystemFault(t *testing.T) {
	// Only reachable if the middleware chain is mis-wired, so it must not read as a
	// client error.
	store := seededStore()
	svc := newService(store, newFakeProjector())

	_, err := svc.List(context.Background())
	assertAOH(t, err, aoherr.ClassSystem, auth.CodeIdentityMissing)
	assert.Empty(t, store.callers, "the store must not be reached")
}

// --------------------------------------------------------------------------- //
// Role gating
// --------------------------------------------------------------------------- //

func TestWrites_RequireTheDispatcherRole(t *testing.T) {
	for _, tc := range []struct {
		name string
		call func(*service.UnitService, context.Context) error
	}{
		{"create", func(s *service.UnitService, ctx context.Context) error {
			_, err := s.Create(ctx, validInput())
			return err
		}},
		{"update", func(s *service.UnitService, ctx context.Context) error {
			_, err := s.Update(ctx, "FU-101", 0, validInput())
			return err
		}},
		{"delete", func(s *service.UnitService, ctx context.Context) error {
			return s.Delete(ctx, "FU-101", 0)
		}},
	} {
		t.Run(tc.name+" refused for a viewer", func(t *testing.T) {
			store := seededStore()
			svc := newService(store, newFakeProjector())

			err := tc.call(svc, viewerCtx())
			aohErr := assertAOH(t, err, aoherr.ClassAuthorization, service.CodeUnitForbidden)
			assert.NotEmpty(t, aohErr.Details())
			assert.Empty(t, store.callers, "the store must not be reached")
		})
	}
}

func TestReads_AreRefusedWhenNoApplicationRoleIsHeld(t *testing.T) {
	store := seededStore()
	svc := newService(store, newFakeProjector())

	_, err := svc.List(ctxAs(tenantA, "mallory", "tenant-user"))
	assertAOH(t, err, aoherr.ClassAuthorization, service.CodeUnitForbidden)
	assert.Empty(t, store.callers)
}

func TestReads_AreAllowedForAViewer(t *testing.T) {
	store := seededStore()
	svc := newService(store, newFakeProjector())

	_, err := svc.List(viewerCtx())
	require.NoError(t, err)
	assert.Len(t, store.callers, 1)
}

// --------------------------------------------------------------------------- //
// Position
// --------------------------------------------------------------------------- //

func TestCreate_OutOfRangeCoordinatesNameTheOffendingFields(t *testing.T) {
	store := seededStore()
	svc := newService(store, newFakeProjector())

	in := validInput()
	in.Position = &domain.PositionInput{Lon: -181, Lat: 91}

	_, err := svc.Create(dispatcherCtx(), in)
	aohErr := assertAOH(t, err, aoherr.ClassValidation, service.CodeUnitInvalid)

	fields := map[string]bool{}
	for _, d := range aohErr.Details() {
		fields[d.Field] = true
	}
	assert.True(t, fields["position.lon"], "details name position.lon: %+v", aohErr.Details())
	assert.True(t, fields["position.lat"], "details name position.lat: %+v", aohErr.Details())
	assert.Empty(t, store.callers, "an invalid write must not reach the store")
}

func TestCreate_BoundaryCoordinatesAreAccepted(t *testing.T) {
	for _, tc := range []struct{ lon, lat float64 }{
		{-180, -90}, {180, 90}, {0, 0},
	} {
		store := seededStore()
		svc := newService(store, newFakeProjector())

		in := validInput()
		in.Position = &domain.PositionInput{Lon: tc.lon, Lat: tc.lat}

		_, err := svc.Create(dispatcherCtx(), in)
		require.NoError(t, err, "lon=%v lat=%v", tc.lon, tc.lat)
	}
}

func TestUpdate_OmittedPositionClearsIt(t *testing.T) {
	// A replace replaces. The absent key is what a clear looks like on the wire, and the
	// repo must be asked for exactly that, not left holding the old fix.
	store := seededStore()
	svc := newService(store, newFakeProjector())

	in := validInput()
	in.Position = nil

	_, err := svc.Update(dispatcherCtx(), "FU-101", 3, in)
	require.NoError(t, err)

	require.Len(t, store.updated, 1)
	assert.Nil(t, store.updated[0].Position)
}

// --------------------------------------------------------------------------- //
// Projection hand-off
// --------------------------------------------------------------------------- //

func TestCreate_HandsTheProjectionToTheWorkerAsTheWritingOperator(t *testing.T) {
	store := seededStore()
	store.rows = []projection.Row{{ID: "row-1", UnitCode: "FU-700", Intent: projection.IntentUpsert}}
	proj := newFakeProjector()
	svc := newService(store, proj)

	_, err := svc.Create(dispatcherCtx(), validInput())
	require.NoError(t, err)

	got := proj.next(t)
	assert.Equal(t, store.rows, got.rows)
	assert.Equal(t, "token-for-"+subjectAda, got.token,
		"the projection carries the operator's own bearer, not a service account's")
	assert.False(t, got.expiry.IsZero(), "delivery is bounded by the token's lifetime")
}

func TestCreate_AFailedWriteProjectsNothing(t *testing.T) {
	store := seededStore()
	store.err = errors.New("boom")
	proj := newFakeProjector()
	svc := newService(store, proj)

	_, err := svc.Create(dispatcherCtx(), validInput())
	assertAOH(t, err, aoherr.ClassSystem, service.CodeUnitWriteFailed)
	proj.assertNothingElse(t)
}

func TestList_AViewerIsNeverMadeToPerformAWrite(t *testing.T) {
	// A reader must not trigger the seed, and must not hand anything to the projection
	// worker — the worker would then write to gis-service on the reader's token.
	store := &fakeStore{seeded: false}
	proj := newFakeProjector()
	svc := newService(store, proj)

	_, err := svc.List(viewerCtx())
	require.NoError(t, err)

	assert.Zero(t, store.seedCalls, "a viewer must not trigger the seed")
	proj.assertNothingElse(t)
}

// --------------------------------------------------------------------------- //
// Seeding
// --------------------------------------------------------------------------- //

func TestList_FirstDispatcherRequestSeedsTheTenantBeforeAnswering(t *testing.T) {
	store := &fakeStore{seeded: false}
	proj := newFakeProjector()
	svc := newService(store, proj)

	_, err := svc.List(dispatcherCtx())
	require.NoError(t, err)

	assert.Equal(t, 1, store.seedCalls)
	require.True(t, store.seedDidRun)

	// The seed's rows travel the same path, on the same token, as any other write.
	got := proj.next(t)
	assert.Equal(t, "token-for-"+subjectAda, got.token)
	assert.Len(t, got.rows, 5, "the baseline roster is five units")

	// And it was attributed to the dispatcher who triggered it.
	require.NotEmpty(t, store.callers)
	assert.Equal(t, subjectAda, store.callers[0].Subject)
	assert.Equal(t, tenantA, store.callers[0].TenantID)
}

func TestList_AnAlreadySeededTenantIsNotSeededAgain(t *testing.T) {
	store := seededStore()
	proj := newFakeProjector()
	svc := newService(store, proj)

	_, err := svc.List(dispatcherCtx())
	require.NoError(t, err)

	assert.Zero(t, store.seedCalls, "the marker, not an emptiness check, decides")
	proj.assertNothingElse(t)
}

func TestList_AnEmptyRosterDoesNotResurrectTheSeed(t *testing.T) {
	// Deleting every unit is a legitimate operator action. The marker survives it, so the
	// roster must stay empty.
	store := seededStore()
	store.list = []domain.Unit{}
	proj := newFakeProjector()
	svc := newService(store, proj)

	units, err := svc.List(dispatcherCtx())
	require.NoError(t, err)
	assert.Empty(t, units)
	assert.Zero(t, store.seedCalls)
	proj.assertNothingElse(t)
}

func TestList_ASeedThatCannotCommitFailsTheRequestLoudly(t *testing.T) {
	// Not "log it and serve an empty roster": an empty console with no error is
	// indistinguishable from a tenant nobody has seeded, and from one emptied on purpose.
	store := &fakeStore{seeded: false, seedErr: errors.New("deadlock detected")}
	proj := newFakeProjector()
	svc := newService(store, proj)

	units, err := svc.List(dispatcherCtx())
	require.Error(t, err)
	assert.Nil(t, units)
	assertAOH(t, err, aoherr.ClassSystem, service.CodeTenantSeedFailed)
	proj.assertNothingElse(t)
}

func TestCreate_SeedFailureIsReportedBeforeTheWriteIsAttempted(t *testing.T) {
	store := &fakeStore{seeded: false, isSeedErr: errors.New("connection refused")}
	svc := newService(store, newFakeProjector())

	_, err := svc.Create(dispatcherCtx(), validInput())
	assertAOH(t, err, aoherr.ClassSystem, service.CodeTenantSeedFailed)
	assert.Empty(t, store.created)
}
