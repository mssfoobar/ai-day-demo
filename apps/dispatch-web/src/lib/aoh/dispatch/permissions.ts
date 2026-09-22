/**
 * Role → permission projection.
 *
 * SOURCE OF TRUTH: `compose/iams/init/project-aas/roles.yaml`. These two role names are
 * AAS *tenant* roles, not Keycloak realm roles, and this table is the code-readable form
 * of the same data. Its Go twin is
 * `apps/dispatch-svc/internal/service/permissions.go` — change one and you must change
 * all three.
 *
 * The projection exists because **there is no `active_tenant.permissions` claim**. AAS
 * puts role names in the token and nothing else, so a UI that gates on a resolved
 * permission list hides every control from every legitimate user.
 *
 * This is the console's half only. It decides what to *render*; the service enforces the
 * same rule independently, because a hidden button is not a permission check.
 */

/** One capability the console gates on. Read vs write is the whole boundary. */
export type Permission = 'read' | 'write';

/** Application role names, as declared in roles.yaml. */
export const ROLE_VIEWER = 'dispatch-viewer';
export const ROLE_DISPATCHER = 'dispatch-dispatcher';

/** Either application role is enough to reach the console and the map. */
export const APPLICATION_ROLES: readonly string[] = [ROLE_VIEWER, ROLE_DISPATCHER];

const ROLE_PERMISSIONS: Record<string, readonly Permission[]> = {
	[ROLE_VIEWER]: ['read'],
	[ROLE_DISPATCHER]: ['read', 'write']
};

/**
 * Whether any of the caller's roles grants `permission`.
 *
 * Roles this app does not know — `tenant-user`, `ai-user`, anything another project
 * declares in the same tenant — grant nothing.
 */
export function hasPermission(roles: readonly string[], permission: Permission): boolean {
	return roles.some((role) => ROLE_PERMISSIONS[role]?.includes(permission) ?? false);
}

/** Convenience for the write gate the add / edit / delete controls hang off. */
export function canWrite(roles: readonly string[]): boolean {
	return hasPermission(roles, 'write');
}

/** Convenience for the read gate the console and map routes hang off. */
export function canRead(roles: readonly string[]): boolean {
	return hasPermission(roles, 'read');
}
