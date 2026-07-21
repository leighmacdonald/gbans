import type { Interceptor } from "@connectrpc/connect";
import { createConnectTransport } from "@connectrpc/connect-web";
import { createValidateInterceptor } from "@connectrpc/validate";
import { QueryClient } from "@tanstack/react-query";
import { StorageKey } from "./auth.tsx";
import { emptyOrNullString } from "./util/types.ts";

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

const authInterceptor: Interceptor = (next) => async (req) => {
	try {
		const value = localStorage.getItem(StorageKey.Token);
		if (emptyOrNullString(value)) {
			return await next(req);
		}
		const parsed = JSON.parse(value);
		if (typeof parsed?.token !== "string" || parsed.token.length === 0) {
			localStorage.removeItem(StorageKey.Token);
			return await next(req);
		}
		req.header.set("Authorization", `Bearer ${parsed.token}`);
	} catch {
		localStorage.removeItem(StorageKey.Token);
	}

	return await next(req);
};

export const finalTransport = createConnectTransport({
	baseUrl: `${window.location.protocol}//${window.location.hostname}:${window.location.port}/connect/`,
	useHttpGet: true,
	interceptors: [validateInterceptor, authInterceptor],
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
