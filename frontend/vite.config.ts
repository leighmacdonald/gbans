import react from '@vitejs/plugin-react-swc';
import { defineConfig } from 'vite';
import { createHtmlPlugin } from 'vite-plugin-html';

// https://vitejs.dev/config/
export default defineConfig({
    plugins: [
        react(),
        createHtmlPlugin({
            entry: 'src/index.tsx',
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
