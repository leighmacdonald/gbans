/// <reference types="vite/client" />
import react from '@vitejs/plugin-react-swc';
import { visualizer } from 'rollup-plugin-visualizer';
import { defineConfig, type PluginOption } from 'vite';
import { createHtmlPlugin } from 'vite-plugin-html';

// https://vitejs.dev/config/
export default defineConfig({
    build: {
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
        visualizer() as PluginOption,
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
