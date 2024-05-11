/// <reference types="vite/client" />
import { TanStackRouterVite } from '@tanstack/router-vite-plugin';
import react from '@vitejs/plugin-react-swc';
import { defineConfig } from 'vite';
import { createHtmlPlugin } from 'vite-plugin-html';

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
    server: {
        open: true,
        port: 6007,
        cors: true,
        host: 'gbans.localhost',
        proxy: {
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
                data: {
                    site_name: 'Uncletopia',
                    build_version: 'v0.6.6',
                    discord_link_id: 'caQKCWFMrN',
                    asset_url: ''
                }
            }
        })
    ]
});
