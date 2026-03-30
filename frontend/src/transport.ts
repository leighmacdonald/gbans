import { createConnectTransport } from "@connectrpc/connect-web";

export const finalTransport = createConnectTransport({
	baseUrl: `${window.location.protocol}//${window.location.hostname}:${window.location.port}/connect/`,
	useHttpGet: true,
});
