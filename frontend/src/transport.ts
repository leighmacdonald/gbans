import { createConnectTransport } from "@connectrpc/connect-web";
import { createValidateInterceptor } from "@connectrpc/validate";
import { QueryClient } from "@tanstack/react-query";

export const queryClient = new QueryClient({
	defaultOptions: {
		queries: {
			staleTime: 30_000,
			gcTime: 5 * 60 * 1000,
			retry: 1,
			refetchOnWindowFocus: false,
		},
	},
});

const validateInterceptor = createValidateInterceptor();

export const finalTransport = createConnectTransport({
	baseUrl: `${window.location.protocol}//${window.location.hostname}:${window.location.port}/connect/`,
	useHttpGet: true,
	interceptors: [validateInterceptor],
});

export function removeUndefinedDeep<T>(obj: T): T {
	if (obj === null || typeof obj !== "object") return obj;
	if (Array.isArray(obj)) return obj.map(removeUndefinedDeep) as T;

	const result = {} as T;
	for (const [key, value] of Object.entries(obj)) {
		if (value !== undefined) {
			result[key as keyof T] = removeUndefinedDeep(value);
		}
	}
	return result;
}
