/// <reference types="vite/client" />
import { sentryVitePlugin } from '@sentry/vite-plugin';
import { TanStackRouterVite } from '@tanstack/router-vite-plugin';
import react from '@vitejs/plugin-react-swc';
import { defineConfig } from 'vite';
import { createHtmlPlugin } from 'vite-plugin-html';

// https://vitejs.dev/config/
export default defineConfig({
    base: '/',
    publicDir: 'public',
    build: {
        copyPublicDir: true,
        sourcemap: true,
        rollupOptions: {
            treeshake: 'recommended',
            output: {
                esModule: false,
                manualChunks: {
                    leaflet: ['leaflet'],
                    'react-leaflet': ['react-leaflet'],
                    'date-fns': ['date-fns']
                }
            }
        }
    },

    server: {
        open: true,
        port: 6007,
        cors: true,
        host: 'gbans.localhost',
        proxy: {
            '/discord/oauth': {
                target: 'http://gbans.localhost:6006',
                changeOrigin: true,
                secure: false
            },
            '/patreon/oauth': {
                target: 'http://gbans.localhost:6006',
                changeOrigin: true,
                secure: false
            },
            '/patreon/login': {
                target: 'http://gbans.localhost:6006',
                changeOrigin: true,
                secure: false
            },
            '/auth/callback': {
                target: 'http://gbans.localhost:6006',
                changeOrigin: true,
                secure: false
            },
            '/api': {
                target: 'http://gbans.localhost:6006',
                changeOrigin: true,
                secure: false
            },
            '/asset': {
                target: 'http://gbans.localhost:6006',
                changeOrigin: true,
                secure: false
            },
            '/ws': {
                target: 'http://gbans.localhost:6006',
                changeOrigin: true,
                secure: false,
                ws: true
            }
        }
    },

    plugins: [
        react(),
        TanStackRouterVite(),
        createHtmlPlugin({
            entry: './src/index.tsx',
            template: 'index.html',
            inject: {}
        }),
        sentryVitePlugin({
            telemetry: false,
            org: 'uncletopia',
            project: 'frontend',
            authToken: process.env.SENTRY_AUTH_TOKEN
        })
    ]
});
