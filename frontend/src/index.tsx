import { StrictMode } from 'react';
import { createRoot } from 'react-dom/client';
import '@fontsource/roboto/latin-300.css';
import '@fontsource/roboto/latin-400.css';
import '@fontsource/roboto/latin-500.css';
import '@fontsource/roboto/latin-700.css';
import * as Sentry from '@sentry/react';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { createRouter, RouterProvider } from '@tanstack/react-router';
import { AuthProvider, useAuth } from './auth.tsx';
import { ErrorDetails } from './component/ErrorDetails.tsx';
import { LoadingPlaceholder } from './component/LoadingPlaceholder.tsx';
import { AppError, ErrorCode } from './error.tsx';
import './fonts/tf2build.css';
import { routeTree } from './routeTree.gen';

const queryClient = new QueryClient();

// Create a new router instance
const router = createRouter({
    routeTree,
    defaultPreload: 'intent',
    context: {
        auth: undefined!,
        queryClient
    },
    defaultPendingComponent: LoadingPlaceholder,
    defaultErrorComponent: () => {
        return <ErrorDetails error={new AppError(ErrorCode.Unknown)} />;
    },
    // Since we're using React Query, we don't want loader calls to ever be stale
    // This will ensure that the loader is always called when the route is preloaded or visited
    defaultPreloadStaleTime: 0
});

// Register the router instance for type safety
declare module '@tanstack/react-router' {
    interface Register {
        router: typeof router;
    }
}

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
    // TODO instrumentation for tanstack router, not currently officially supported
    Sentry.init({
        dsn: window.gbans.sentry_dsn_web,
        release: window.gbans.build_version,
        // Performance Monitoring
        tracesSampleRate: 1.0, //  Capture 100% of the transactions
        // Session Replay
        replaysSessionSampleRate: 0.1, // This sets the sample rate at 10%. You may want to change it to 100% while in development and then sample at a lower rate in production.
        replaysOnErrorSampleRate: 1.0 // If you're not already sampling the entire session, change the sample rate to 100% when sampling sessions where errors occur.
    });
}

function InnerApp() {
    const auth = useAuth();
    return <RouterProvider defaultPreload={'intent'} router={router} context={{ auth }} />;
}

function App() {
    return (
        <AuthProvider>
            <QueryClientProvider client={queryClient}>
                <StrictMode>
                    <InnerApp />
                </StrictMode>
            </QueryClientProvider>
        </AuthProvider>
    );
}

const container = document.getElementById('root');
if (container) {
    createRoot(container).render(<App />);
}
