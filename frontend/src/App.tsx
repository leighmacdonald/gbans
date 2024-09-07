import { PropsWithChildren, StrictMode, useState } from 'react';
import { QueryClient, QueryClientProvider, useQuery } from '@tanstack/react-query';
import { createRouter, RouterProvider } from '@tanstack/react-router';
import { isBefore, parseISO } from 'date-fns';
import { defaultAvatarHash, PermissionLevel } from './api';
import { appInfoDetail, getAppInfo } from './api/app.ts';
import { AuthProvider, profileKey } from './auth.tsx';
import { ErrorDetails } from './component/ErrorDetails.tsx';
import { LoadingPlaceholder } from './component/LoadingPlaceholder.tsx';
import { UseAppInfoCtx } from './contexts/AppInfoCtx.ts';
import { AppError, ErrorCode } from './error.tsx';
import { useAuth } from './hooks/useAuth.ts';
import { routeTree } from './routeTree.gen.ts';
import { logErr } from './util/errors.ts';

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
        logErr(e);
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
        patreon_client_id: '',
        discord_client_id: '',
        default_route: '/',
        discord_enabled: false,
        patreon_enabled: false,
        servers_enabled: false,
        wiki_enabled: false,
        forums_enabled: false,
        stats_enabled: false,
        reports_enabled: false,
        contests_enabled: false,
        chatlogs_enabled: false,
        demos_enabled: false,
        news_enabled: false
    });

    useQuery({
        queryKey: ['appInfo'],
        queryFn: async () => {
            const appInfoString = localStorage.getItem('appInfo');
            const appInfoValidUntil = localStorage.getItem('appInfoValidUntil');
            if (appInfoValidUntil && appInfoString) {
                try {
                    const validDate = parseISO(appInfoValidUntil);
                    if (isBefore(validDate, new Date())) {
                        const cached = JSON.parse(appInfoString) as appInfoDetail;
                        setAppInfo(cached);
                        return cached;
                    }
                } catch (e) {
                    logErr(`Failed to parse appInfo: ${e}`);
                }
            }

            const details = await getAppInfo();
            setAppInfo(details);
            localStorage.setItem('appInfo', JSON.stringify(details));

            const expiry = new Date();
            expiry.setDate(expiry.getHours() + 1);
            localStorage.setItem('appInfoValidUntil', JSON.stringify(expiry));

            return details;
        }
    });

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
