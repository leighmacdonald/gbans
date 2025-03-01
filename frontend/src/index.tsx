import { createRoot } from 'react-dom/client';
import '@fontsource/roboto/latin-300.css';
import '@fontsource/roboto/latin-400.css';
import '@fontsource/roboto/latin-500.css';
import '@fontsource/roboto/latin-700.css';
import * as Sentry from '@sentry/react';
import { QueryClient } from '@tanstack/react-query';
import { App } from './App.tsx';
import './fonts/tf2build.css';
import { newRouter } from './router.tsx';

const queryClient = new QueryClient();
const router = newRouter(queryClient);

// Register the router instance for type safety
declare module '@tanstack/react-router' {
    // noinspection JSUnusedGlobalSymbols
    interface Register {
        router: typeof router;
    }
}

if (import.meta.env.VITE_SENTRY_DSN != '') {
    const target = `^https://${window.location.origin}/api`;

    // TODO instrumentation for tanstack router, not currently officially supported
    Sentry.init({
        environment: import.meta.env.MODE,
        attachStacktrace: true,
        dsn: import.meta.env.VITE_SENTRY_DSN,
        release: import.meta.env.VITE_BUILD_VERSION,
        integrations: [
            Sentry.tanstackRouterBrowserTracingIntegration(router),
            Sentry.browserTracingIntegration(),
            Sentry.browserProfilingIntegration(),
            Sentry.replayIntegration({
                maskAllText: false,
                blockAllMedia: false
            })
        ],
        // Performance Monitoring
        tracesSampleRate: 1.0, //  Capture 100% of the transactions
        tracePropagationTargets: ['localhost', target],
        // Session Replay
        replaysSessionSampleRate: 0.1, // This sets the sample rate at 10%. You may want to change it to 100% while in development and then sample at a lower rate in production.
        replaysOnErrorSampleRate: 1.0 // If you're not already sampling the entire session, change the sample rate to 100% when sampling sessions where errors occur.
    });
}

const AppProfiler = Sentry.withProfiler(App, { name: 'gbans' });

const container = document.getElementById('root');
if (container) {
    if (import.meta.env.VITE_SENTRY_DSN != '') {
        createRoot(container).render(<AppProfiler queryClient={queryClient} router={router} />);
    } else {
        createRoot(container).render(<App queryClient={queryClient} router={router} />);
    }
}
