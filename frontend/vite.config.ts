/// <reference types="vite/client" />
import { tanstackRouter } from "@tanstack/router-vite-plugin";
import react from "@vitejs/plugin-react";
import { defineConfig } from "vite";
import { createHtmlPlugin } from "vite-plugin-html";
import { bgWebPPlugin } from "./vite-bg-plugin.ts";

// https://vitejs.dev/config/
export default defineConfig({
	base: "",
	publicDir: "public",
	build: {
		copyPublicDir: true,
		sourcemap: false,
		target: "esnext",
		chunkSizeWarningLimit: 1000,
		rolldownOptions: {
			checks: {
				circularDependency: true,
			},
			output: {
				manualChunks(id) {
					if (id.includes("@mui/x-charts")) return "mui-charts";
					if (id.includes("leaflet") || id.includes("react-leaflet")) return "leaflet";
					if (id.includes("@mdxeditor/editor")) return "mdx-editor";
					if (id.includes("react-player")) return "react-player";
					if (id.includes("@tanstack/react-form")) return "react-form";
					if (id.includes("zod/")) return "zod";
				},
			},
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
			"/rpc": {
				target: "http://gbans.localhost:6006",
				changeOrigin: true,
				secure: false,
			},
			"/asset": {
				target: "http://gbans.localhost:6006",
				changeOrigin: true,
				secure: false,
			},
		},
	},

	plugins: [
		createHtmlPlugin({
			entry: "./src/index.tsx",
			template: "index.html",
			inject: {},
		}),
		tanstackRouter({
			target: "react",
			autoCodeSplitting: true,
		}),
		react(), // Must come *after* tanstackRouter
		bgWebPPlugin(),
	],
});
