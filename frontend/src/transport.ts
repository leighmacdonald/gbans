import { createConnectTransport } from "@connectrpc/connect-web";
import type { Interceptor } from "@connectrpc/connect";
import { accessTokenKey } from "./auth.tsx";
import { emptyOrNullString } from "./util/types.ts";

const authInterceptor: Interceptor = (next) => async (req) => {
	const token = localStorage.getItem(accessTokenKey);
	if (!emptyOrNullString(token)) {
		req.header.set("Authorization", "Bearer " + token);
	}

	return await next(req);
};

export const finalTransport = createConnectTransport({
	baseUrl: `${window.location.protocol}//${window.location.hostname}:${window.location.port}/connect/`,
	useHttpGet: true,
	interceptors: [authInterceptor],
});
