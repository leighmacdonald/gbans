import { createRoot } from 'react-dom/client';
import '@fontsource/roboto/latin-300.css';
import '@fontsource/roboto/latin-400.css';
import '@fontsource/roboto/latin-500.css';
import '@fontsource/roboto/latin-700.css';
import * as Sentry from '@sentry/react';
import { App } from './App.tsx';
import './fonts/tf2build.css';

// extend window with our own items that we inject
declare global {
    interface Window {
        gbans: {
            discord_client_id: string;
            build_date: string;
            build_commit: string;
            sentry_dsn_web: string;
        };
    }
}

window.gbans = window.gbans || [];

if (window.gbans.sentry_dsn_web != '') {
    // TODO instrumentation for tanstack router, not currently officially supported
    Sentry.init({
        dsn: window.gbans.sentry_dsn_web,
        release: __BUILD_VERSION__,
        // Performance Monitoring
        tracesSampleRate: 1.0, //  Capture 100% of the transactions
        // Session Replay
        replaysSessionSampleRate: 0.1, // This sets the sample rate at 10%. You may want to change it to 100% while in development and then sample at a lower rate in production.
        replaysOnErrorSampleRate: 1.0 // If you're not already sampling the entire session, change the sample rate to 100% when sampling sessions where errors occur.
    });
}

const container = document.getElementById('root');
if (container) {
    createRoot(container).render(<App />);
}
