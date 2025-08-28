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
                //esModule: false,
                manualChunks(id) {
                    const chunks = [
                        'sentry',
                        'react-leaflet',
                        'icons-material',
                        'leaflet',
                        'nice-modal-react',
                        'fontsource/roboto',
                        'mdxeditor/editor',
                        'prism-react-renderer',
                        'mui-markdown',
                        'date-fns',
                        'mui/x-charts',
                        'mui/x-date-pickers',
                        'mui/lab',
                        'emotion',
                        'mui/material',
                        'mui/system',
                        'mui/utils',
                        'tanstack/react-form',
                        'tanstack/react-query',
                        'tanstack/react-router',
                        'tanstack/react-table',
                        'core-js',
                        'eslint',
                        'markdown-to-jsx',
                        'mui-markdown',
                        'mui-image',
                        'material-ui-popup-state',
                        'minimatch',
                        'zod',
                        'video-react',
                        'typescript',
                        'steamid',
                        'js-cookie',
                        'file-type',
                        'ip-cidr',
                        'base64-js',
                        'mui-nested-menu',
                        'react',
                        'react-modal-image',
                        'react-scrollable-feed',
                        'react-timer-hook',
                        'react-use-websocket'
                    ];

                    if (id.includes('node_modules')) {
                        return (chunks.find((c) => id.includes(c)) ?? 'vendor').replace('/', '-');
                    }

                    if (id.includes('modal')) {
                        return 'modal';
                    }

                    if (id.includes('.png')) {
                        return 'pngs';
                    }

                    return null;
                }
            }
        }
    },

    server: {
        open: true,
        port: 6007,
        cors: true,
        host: '0.0.0.0',
        allowedHosts: true, // WARN You should set __VITE_ADDITIONAL_SERVER_ALLOWED_HOSTS instead
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
            authToken: import.meta.env?.SENTRY_AUTH_TOKEN ?? ''
        })
    ]
});
