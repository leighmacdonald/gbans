import { PropsWithChildren, StrictMode, useEffect, useState } from 'react';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { createRouter, RouterProvider } from '@tanstack/react-router';
import { apiCall, appInfoDetail, defaultAvatarHash, PermissionLevel } from './api';
import { AuthProvider, profileKey } from './auth.tsx';
import { ErrorDetails } from './component/ErrorDetails.tsx';
import { LoadingPlaceholder } from './component/LoadingPlaceholder.tsx';
import { UseAppInfoCtx } from './contexts/AppInfoCtx.ts';
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

const loadProfile = () => {
    const defaultProfile = {
        steam_id: '',
        permission_level: PermissionLevel.Guest,
        avatarhash: defaultAvatarHash,
        name: '',
        ban_id: 0,
        muted: false,
        discord_id: '',
        created_on: new Date(),
        updated_on: new Date()
    };
    try {
        const userData = localStorage.getItem(profileKey);
        if (!userData) {
            return defaultProfile;
        }

        return JSON.parse(userData);
    } catch (e) {
        return defaultProfile;
    }
};

export function App() {
    const [profile, setProfile] = useState(loadProfile());

    return (
        <AuthProvider profile={profile} setProfile={setProfile}>
            <QueryClientProvider client={queryClient}>
                <AppInfoProvider>
                    <StrictMode>
                        <InnerApp />
                    </StrictMode>
                </AppInfoProvider>
            </QueryClientProvider>
        </AuthProvider>
    );
}

const InnerApp = () => {
    const auth = useAuth();

    return <RouterProvider defaultPreload={'intent'} router={router} context={{ auth }} />;
};

const AppInfoProvider = ({ children }: PropsWithChildren) => {
    const [appInfo, setAppInfo] = useState<appInfoDetail>({
        app_version: '',
        link_id: '',
        sentry_dns_web: '',
        site_name: 'Loading',
        asset_url: '/assets',
        patreon_client_id: ''
    });

    useEffect(() => {
        apiCall<appInfoDetail>('/api/info')
            .then((value) => {
                setAppInfo(value);
            })
            .catch((reason) => {
                console.log(reason);
            });
    }, []);

    return (
        <UseAppInfoCtx.Provider
            value={{
                setAppInfo,
                appInfo
            }}
        >
            {children}
        </UseAppInfoCtx.Provider>
    );
};
