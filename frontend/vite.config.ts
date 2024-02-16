/// <reference types="vite/client" />
import react from '@vitejs/plugin-react-swc';
import { defineConfig } from 'vite';
import { createHtmlPlugin } from 'vite-plugin-html';

// https://vitejs.dev/config/
export default defineConfig({
    build: {
        //sourcemap: true,
        rollupOptions: {
            output: {
                manualChunks: {
                    leaflet: ['leaflet'],
                    'react-leaflet': ['react-leaflet'],
                    'date-fns': ['date-fns']
                }
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
