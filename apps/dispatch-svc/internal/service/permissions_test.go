package service_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/mssfoobar/fleet-dispatch-console/apps/dispatch-svc/internal/service"
)

// The projection mirrors compose/iams/init/project-aas/roles.yaml. These cases are the
// whole matrix, so a change to roles.yaml that is not mirrored here fails loudly.
func TestHasPermission_RoleMatrix(t *testing.T) {
	cases := []struct {
		name  string
		roles []string
		read  bool
		write bool
	}{
		{"dispatcher reads and writes", []string{service.RoleDispatcher}, true, true},
		{"viewer reads only", []string{service.RoleViewer}, true, false},
		{"both roles read and write", []string{service.RoleViewer, service.RoleDispatcher}, true, true},

		// A real token from the seeded realm carries platform and other projects' roles
		// alongside ours. They must grant nothing here.
		{"platform roles alone grant nothing", []string{"tenant-user", "ai-user"}, false, false},
		{"viewer among unrelated roles still reads", []string{"tenant-user", service.RoleViewer}, true, false},
		{"no roles at all", nil, false, false},
		{"unknown role", []string{"dispatch-admin"}, false, false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.read, service.HasPermission(tc.roles, service.PermissionRead), "read")
			assert.Equal(t, tc.write, service.HasPermission(tc.roles, service.PermissionWrite), "write")
		})
	}
}

// There is no `active_tenant.permissions` claim — AAS does not emit one. This pins that
// the projection's inputs are role names, so nobody "fixes" it by reading a permissions
// list that will always be absent.
func TestHasPermission_PermissionNamesAreNotRoleNames(t *testing.T) {
	assert.False(t, service.HasPermission([]string{string(service.PermissionWrite)}, service.PermissionWrite))
	assert.False(t, service.HasPermission([]string{string(service.PermissionRead)}, service.PermissionRead))
}
