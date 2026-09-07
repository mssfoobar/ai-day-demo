export type ModuleConfig = {
	host: string;
	basePath?: string;
	getHeaders?: (context: {
		accessToken: string;
		requestHeaders: Headers;
	}) => Promise<Headers> | Headers;
};

export type GatewayConfig = {
	modules: Record<string, ModuleConfig>;
};

export const gatewayConfig: GatewayConfig = {
	// Map module codes to backend services. Each entry is reachable at
	// `/aoh/gateway/<code>/...` via src/routes/(private)/aoh/gateway/[...path]/+server.ts.
	// Example:
	//   myservice: {
	//     host: process.env.MY_SERVICE_URL || 'http://my-service:8080'
	//   }
	modules: {}
};

export function resolveModule(modulePath: string): { url: string; config: ModuleConfig } | null {
	const [moduleCode, ...restPath] = modulePath.split('/');
	const moduleConfig = gatewayConfig.modules[moduleCode];
	if (!moduleConfig) return null;
	const basePath = moduleConfig.basePath || '';
	const remainingPath = restPath.join('/');
	return {
		url: `${moduleConfig.host}${basePath}/${remainingPath}`,
		config: moduleConfig
	};
}

export function resolveModuleUrl(modulePath: string): string | null {
	return resolveModule(modulePath)?.url ?? null;
}
