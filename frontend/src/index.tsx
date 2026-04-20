import { createRoot, type Root } from "react-dom/client";
import "@fontsource/roboto/latin-300.css";
import "@fontsource/roboto/latin-400.css";
import "@fontsource/roboto/latin-500.css";
import "@fontsource/roboto/latin-700.css";
import * as Sentry from "@sentry/react";
import { QueryClient } from "@tanstack/react-query";
import { App } from "./App.tsx";
import "./fonts/tf2build.css";
import { createClient } from "@connectrpc/connect";
import { createConnectQueryKey } from "@connectrpc/connect-query";
import { ConfigService } from "./rpc/config/v1/config_pb.ts";
import { newRouter } from "./router.tsx";
import { finalTransport } from "./transport.ts";

// Register the router instance for type safety
declare module "@tanstack/react-router" {
	// noinspection JSUnusedGlobalSymbols
	interface Register {
		router: typeof router;
	}
}

const queryClient = new QueryClient();
const configClient = createClient(ConfigService, finalTransport);

const appInfo = await queryClient.fetchQuery({
	queryKey: createConnectQueryKey({
		schema: ConfigService,
		transport: finalTransport,
		cardinality: "finite",
	}),
	queryFn: async () => {
		return await configClient.info({});
	},
});
const router = newRouter(queryClient, appInfo);
const container = document.getElementById("root");
if (!container) {
	throw new Error("Root element not found");
}

let root: Root;
if (appInfo.sentryDsnWeb !== "") {
	Sentry.init({
		environment: import.meta.env.MODE,
		attachStacktrace: true,
		enableLogs: false,
		dsn: appInfo.sentryDsnWeb,
		release: import.meta.env.VITE_BUILD_VERSION,
		integrations: [Sentry.tanstackRouterBrowserTracingIntegration(router)],
	});
	root = createRoot(container, {
		onUncaughtError: Sentry.reactErrorHandler((error, errorInfo) => {
			console.error("Uncaught error", error, errorInfo.componentStack);
		}),
		onCaughtError: Sentry.reactErrorHandler(),
		onRecoverableError: Sentry.reactErrorHandler(),
	});
} else {
	root = createRoot(container);
}
root.render(<App queryClient={queryClient} router={router} />);
