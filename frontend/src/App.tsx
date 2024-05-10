import { StrictMode } from 'react';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { createRouter, RouterProvider } from '@tanstack/react-router';
import { AuthProvider } from './auth.tsx';
import { ErrorDetails } from './component/ErrorDetails.tsx';
import { LoadingPlaceholder } from './component/LoadingPlaceholder.tsx';
import { AppError, ErrorCode } from './error.tsx';
import { useAuth } from './hooks/useAuth.ts';
import { routeTree } from './routeTree.gen.ts';

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
    // noinspection JSUnusedGlobalSymbols
    interface Register {
        router: typeof router;
    }
}

export function App() {
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

const InnerApp = () => {
    const auth = useAuth();
    return <RouterProvider defaultPreload={'intent'} router={router} context={{ auth }} />;
};
