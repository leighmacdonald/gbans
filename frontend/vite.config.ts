/// <reference types="vite/client" />
import { TanStackRouterVite } from '@tanstack/router-vite-plugin';
import react from '@vitejs/plugin-react-swc';
import { defineConfig } from 'vite';
import { createHtmlPlugin } from 'vite-plugin-html';

const CONFIG = {
    __BUILD_VERSION__: 'v0.7.8'
};

const mapValues = (o: object, fn: (v: unknown) => unknown) =>
    Object.fromEntries(Object.entries(o).map(([k, v]) => [k, fn(v)]));

// https://vitejs.dev/config/
export default defineConfig({
    base: '/',
    build: {
        //sourcemap: true,
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

    // https://vitejs.dev/config/shared-options.html#define
    define: mapValues(CONFIG, JSON.stringify),

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
            }
        }
    },

    plugins: [
        react(),
        TanStackRouterVite(),
        createHtmlPlugin({
            entry: './src/index.tsx',
            template: 'index.html',

            inject: {
                data: CONFIG
            }
        })
    ]
});
