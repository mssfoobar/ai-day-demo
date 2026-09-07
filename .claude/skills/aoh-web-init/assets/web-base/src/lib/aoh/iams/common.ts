import type { AuthClaims } from '$lib/aoh/core/provider/auth/auth';

export function hasRole(claims: AuthClaims | undefined, role: string): boolean {
	if (!claims) return false;
	if (claims.realm_access?.roles?.includes(role)) return true;
	if (claims.active_tenant?.roles?.includes(role)) return true;
	return false;
}

export function getSub(claims: AuthClaims | undefined): string | undefined {
	return claims?.sub;
}

export function getActiveTenantId(claims: AuthClaims | undefined): string | undefined {
	return claims?.active_tenant?.tenant_id;
}
