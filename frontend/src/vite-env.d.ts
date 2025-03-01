/// <reference types="vite/client" />

interface ImportMetaEnv {
    readonly VITE_SENTRY_DSN: string;
    readonly VITE_BUILD_VERSION: string;
}

interface ImportMeta {
    readonly env: ImportMetaEnv;
}
