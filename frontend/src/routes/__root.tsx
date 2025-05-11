import { Fragment, PropsWithChildren, useCallback, useMemo, useState } from 'react';
import NiceModal from '@ebay/nice-modal-react';
import { PaletteMode } from '@mui/material';
import { AlertColor } from '@mui/material/Alert';
import Container from '@mui/material/Container';
import CssBaseline from '@mui/material/CssBaseline';
import { ThemeProvider } from '@mui/material/styles';
import useMediaQuery from '@mui/material/useMediaQuery';
import { AdapterDateFns } from '@mui/x-date-pickers/AdapterDateFns';
import { LocalizationProvider } from '@mui/x-date-pickers/LocalizationProvider';
import * as Sentry from '@sentry/react';
import { QueryClient } from '@tanstack/react-query';
import { createRootRouteWithContext, Outlet } from '@tanstack/react-router';
import { PermissionLevel } from '../api';
import { AuthContextProps } from '../auth.tsx';
import { BackgroundImageProvider } from '../component/BackgroundImageProvider.tsx';
import { Flash, Flashes } from '../component/Flashes.tsx';
import { Footer } from '../component/Footer.tsx';
import { LogoutHandler } from '../component/LogoutHandler.tsx';
import { NotificationsProvider } from '../component/NotificationsProvider.tsx';
import { TopBar } from '../component/TopBar.tsx';
import { QueueChat } from '../component/queue/QueueChat.tsx';
import { QueueProvider } from '../component/queue/QueueProvider.tsx';
import { ColourModeContext } from '../contexts/ColourModeContext.tsx';
import { UserFlashCtx } from '../contexts/UserFlashCtx.tsx';
import { ApiError, isApiError } from '../error.tsx';
import { useAuth } from '../hooks/useAuth.ts';
import { createThemeByMode } from '../theme.ts';
import { checkFeatureEnabled } from '../util/features.ts';
import { emptyOrNullString } from '../util/types.ts';

type RouterContext = {
    auth: AuthContextProps;
    queryClient: QueryClient;
};

export const Route = createRootRouteWithContext<RouterContext>()({
    component: Root
});

const OptionalQueueProvider = ({ children }: PropsWithChildren) => {
    const { isAuthenticated } = useAuth();
    if (isAuthenticated()) {
        return <QueueProvider>{children}</QueueProvider>;
    }
    return children;
};

function Root() {
    const initialTheme = (localStorage.getItem('theme') as PaletteMode) || 'light';
    const { hasPermission } = useAuth();
    const [flashes, setFlashes] = useState<Flash[]>([]);
    const prefersDarkMode = useMediaQuery('(prefers-color-scheme: dark)');
    const [mode, setMode] = useState<'light' | 'dark'>(
        initialTheme ? initialTheme : prefersDarkMode ? 'dark' : 'light'
    );

    const updateMode = (prevMode: PaletteMode): PaletteMode => {
        const m = prevMode === 'light' ? 'dark' : ('light' as PaletteMode);
        localStorage.setItem('theme', m);
        return m;
    };

    const colorMode = useMemo(
        () => ({
            toggleColorMode: () => {
                setMode(updateMode);
            }
        }),
        []
    );

    const theme = useMemo(() => createThemeByMode(mode), [mode]);

    const sendFlash = useCallback(
        (level: AlertColor, message: string, heading = '', closable = true) => {
            if (flashes.length && flashes[flashes.length - 1]?.message == message) {
                // Skip duplicates
                return;
            }
            if (emptyOrNullString(heading)) {
                heading = level;
            }
            setFlashes([
                ...flashes,
                {
                    closable: closable ?? false,
                    heading: heading,
                    level: level,
                    message: message
                }
            ]);
        },
        [flashes, setFlashes]
    );

    const sendError = useCallback(
        (error: unknown) => {
            if (!error) {
                return;
            }
            if (isApiError(error)) {
                sendFlash('error', (error as ApiError).detail, (error as ApiError).title, true);
            } else {
                sendFlash('error', (error as Error).message, (error as Error).name, true);
            }

            Sentry.captureException(error);
        },
        [sendFlash]
    );

    return (
        <UserFlashCtx.Provider value={{ flashes, setFlashes, sendFlash, sendError }}>
            <OptionalQueueProvider>
                <LocalizationProvider dateAdapter={AdapterDateFns}>
                    <Fragment>
                        <ColourModeContext.Provider value={colorMode}>
                            <ThemeProvider theme={theme}>
                                <BackgroundImageProvider />
                                <NotificationsProvider>
                                    <NiceModal.Provider>
                                        <LogoutHandler />
                                        <CssBaseline />

                                        <Container maxWidth={'lg'}>
                                            <TopBar />
                                            <div
                                                style={{
                                                    marginTop: 24
                                                }}
                                            >
                                                {checkFeatureEnabled('playerqueue_enabled') &&
                                                    hasPermission(PermissionLevel.Moderator) && <QueueChat />}
                                                <Outlet />
                                            </div>
                                            <Footer />
                                        </Container>
                                        <Flashes />
                                    </NiceModal.Provider>
                                </NotificationsProvider>
                            </ThemeProvider>
                        </ColourModeContext.Provider>
                    </Fragment>
                </LocalizationProvider>
            </OptionalQueueProvider>
        </UserFlashCtx.Provider>
    );
}
