/// <reference types="vite/client" />
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
        proxy: {
            '/auth/callback': {
                target: 'http://localhost:6006',
                changeOrigin: true,
                secure: false
            },
            '/api': {
                target: 'http://localhost:6006',
                changeOrigin: true,
                secure: false
            }
        }
    },

    plugins: [
        react(),
        createHtmlPlugin({
            entry: './src/index.tsx',
            template: 'index.html',

            inject: {
                data: {
                    title: 'gbans',
                    build_version: 'v0.5.14'
                }
            }
        })
    ]
});
