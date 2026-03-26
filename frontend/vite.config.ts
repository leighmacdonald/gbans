/// <reference types="vite/client" />
import { tanstackRouter } from "@tanstack/router-vite-plugin";
import react from "@vitejs/plugin-react";
import { defineConfig } from "vite";
import { createHtmlPlugin } from "vite-plugin-html";

// https://vitejs.dev/config/
export default defineConfig({
	base: "/",
	publicDir: "public",
	legacy: {
		// Required until react-video is updated: https://github.com/robtaussig/react-use-websocket/issues/280
		inconsistentCjsInterop: true,
	},
	build: {
		copyPublicDir: true,
		sourcemap: true,
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
	],
});
