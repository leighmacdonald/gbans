/// <reference types="vite/client" />
import { sentryVitePlugin } from "@sentry/vite-plugin";
import { tanstackRouter } from "@tanstack/router-vite-plugin";
import react from "@vitejs/plugin-react-swc";
import { defineConfig } from "vite";
import { createHtmlPlugin } from "vite-plugin-html";

// https://vitejs.dev/config/
export default defineConfig({
	base: "/",
	publicDir: "public",
	build: {
		copyPublicDir: true,
		sourcemap: true,
		rollupOptions: {
			treeshake: "smallest",
			// output: {
			// 	//esModule: false,
			// 	manualChunks(id) {
			// 		if (id.includes("node_modules")) {
			// 			return "vendor";
			// 		}
			// 		return null;
			// 	},
			// },
		},
	},

	server: {
		open: false,
		port: 6007,
		cors: true,
		host: "0.0.0.0",
		allowedHosts: true, // WARN You should set __VITE_ADDITIONAL_SERVER_ALLOWED_HOSTS instead
		proxy: {
			"/connect": {
				target: "http://gbans.localhost:6006",
				changeOrigin: true,
				secure: false,
			},
			"/discord/oauth": {
				target: "http://gbans.localhost:6006",
				changeOrigin: true,
				secure: false,
			},
			"/patreon/oauth": {
				target: "http://gbans.localhost:6006",
				changeOrigin: true,
				secure: false,
			},
			"/patreon/login": {
				target: "http://gbans.localhost:6006",
				changeOrigin: true,
				secure: false,
			},
			"/auth/callback": {
				target: "http://gbans.localhost:6006",
				changeOrigin: true,
				secure: false,
			},
			"/api": {
				target: "http://gbans.localhost:6006",
				changeOrigin: true,
				secure: false,
			},
			"/asset": {
				target: "http://gbans.localhost:6006",
				changeOrigin: true,
				secure: false,
			},
			"/ws": {
				target: "http://gbans.localhost:6006",
				changeOrigin: true,
				secure: false,
				ws: true,
			},
		},
	},

	plugins: [
		tanstackRouter({
			target: "react",
			autoCodeSplitting: true,
		}),
		react(), // Must come *after* tanstackRouter
		createHtmlPlugin({
			entry: "./src/index.tsx",
			template: "index.html",
			inject: {},
		}),
		sentryVitePlugin({
			telemetry: false,
			org: "uncletopia",
			project: "frontend",
			authToken: import.meta.env?.SENTRY_AUTH_TOKEN ?? "",
		}),
	],
});
