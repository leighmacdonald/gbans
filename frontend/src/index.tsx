import { createRoot, type Root } from "react-dom/client";
import "@fontsource/roboto/latin-300.css";
import "@fontsource/roboto/latin-400.css";
import "@fontsource/roboto/latin-500.css";
import "@fontsource/roboto/latin-700.css";
import * as Sentry from "@sentry/react";
import { QueryClient } from "@tanstack/react-query";
import { App } from "./App.tsx";
import "./fonts/tf2build.css";
import { getAppInfo } from "./api/app.ts";
import { newRouter } from "./router.tsx";

// Register the router instance for type safety
declare module "@tanstack/react-router" {
	// noinspection JSUnusedGlobalSymbols
	interface Register {
		router: typeof router;
	}
}

const queryClient = new QueryClient();
const appInfo = await queryClient.fetchQuery({
	queryKey: ["appInfo"],
	queryFn: getAppInfo,
});
const router = newRouter(queryClient, appInfo);
const container = document.getElementById("root");

let root: Root;
if (container) {
	if (appInfo.sentry_dns_web !== "") {
		Sentry.init({
			environment: import.meta.env.MODE,
			attachStacktrace: true,
			enableLogs: false,
			dsn: appInfo.sentry_dns_web,
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
}
