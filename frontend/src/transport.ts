import { createValidator } from "@bufbuild/protovalidate";
import type { Interceptor } from "@connectrpc/connect";
import { createConnectTransport } from "@connectrpc/connect-web";
import { createValidateInterceptor } from "@connectrpc/validate";
import { QueryClient } from "@tanstack/react-query";
import { StorageKey } from "./auth.tsx";

export const queryClient = new QueryClient();

const validateInterceptor = createValidateInterceptor({ validator: createValidator({}) });

const authInterceptor: Interceptor = (next) => async (req) => {
	try {
		const token = localStorage.getItem(StorageKey.Token) as { token?: string };

		req.header.set("Authorization", `Bearer ${token.token}`);
		console.log(token.token);
	} catch {
		console.log("Failed to load token");
	}

	return await next(req);
};

export const finalTransport = createConnectTransport({
	baseUrl: `${window.location.protocol}//${window.location.hostname}:${window.location.port}/connect/`,
	useHttpGet: true,
	interceptors: [validateInterceptor, authInterceptor],
});
