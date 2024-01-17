import React from 'react';
import { createRoot } from 'react-dom/client';
import { PaletteMode } from '@mui/material';
import * as Sentry from '@sentry/react';
import { App } from './App';

// extend window with our own items that we inject
declare global {
    interface Window {
        gbans: {
            site_name: string;
            discord_client_id: string;
            discord_link_id: string;
            asset_url: string;
            bucket_media: string;
            bucket_demo: string;
            build_version: string;
            build_date: string;
            build_commit: string;
            sentry_dsn_web: string;
        };
    }
}

window.gbans = window.gbans || [];

if (window.gbans.sentry_dsn_web != '') {
    Sentry.init({
        dsn: window.gbans.sentry_dsn_web,
        integrations: [
            new Sentry.BrowserTracing({
                // Set 'tracePropagationTargets' to control for which URLs distributed tracing should be enabled
                tracePropagationTargets: [
                    'localhost',
                    /^https:\/\/yourserver\.io\/api/
                ]
            }),
            new Sentry.Replay({
                maskAllText: false,
                blockAllMedia: false
            })
        ],
        // Performance Monitoring
        tracesSampleRate: 1.0, //  Capture 100% of the transactions
        // Session Replay
        replaysSessionSampleRate: 0.1, // This sets the sample rate at 10%. You may want to change it to 100% while in development and then sample at a lower rate in production.
        replaysOnErrorSampleRate: 1.0 // If you're not already sampling the entire session, change the sample rate to 100% when sampling sessions where errors occur.
    });
}

const container = document.getElementById('root');
if (container) {
    createRoot(container).render(
        <App
            initialTheme={
                (localStorage.getItem('theme') as PaletteMode) || 'light'
            }
        />
    );
}
