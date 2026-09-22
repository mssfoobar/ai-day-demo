package service

// Role → permission projection.
//
// SOURCE OF TRUTH: compose/iams/init/project-aas/roles.yaml. These two role names are
// AAS *tenant* roles, not Keycloak realm roles, and this table is the code-readable form
// of the same data. Its TypeScript twin is
// apps/dispatch-web/src/lib/aoh/dispatch/permissions.ts — change one and you must change
// all three (design.md R3).
//
// The projection exists because **there is no `active_tenant.permissions` claim**. AAS
// puts role names in the token and nothing else; middleware that gates on a resolved
// permission list 403s every legitimate user. So the gate reads `active_tenant.roles` and
// resolves it here, in-process (design.md D3).
//
// The alternative — a live AAS `/evaluate` call per request — is authoritative but puts a
// network hop on every call for a two-role matrix, and puts AAS on this service's request
// path. Revisit that if the matrix grows past a handful of roles.

// Permission is one capability the service gates on. Read vs write is the whole boundary.
type Permission string

const (
	// PermissionRead — list and fetch units.
	PermissionRead Permission = "read"
	// PermissionWrite — create, replace and delete units.
	PermissionWrite Permission = "write"
)

// Application role names, as declared in roles.yaml.
const (
	RoleViewer     = "dispatch-viewer"
	RoleDispatcher = "dispatch-dispatcher"
)

var rolePermissions = map[string][]Permission{
	RoleViewer:     {PermissionRead},
	RoleDispatcher: {PermissionRead, PermissionWrite},
}

// HasPermission reports whether any of the caller's roles grants p.
//
// Roles this service does not know — `tenant-user`, `ai-user`, anything another project
// declares in the same tenant — grant nothing, so a token carrying only those is refused.
func HasPermission(roles []string, p Permission) bool {
	for _, role := range roles {
		for _, granted := range rolePermissions[role] {
			if granted == p {
				return true
			}
		}
	}
	return false
}
