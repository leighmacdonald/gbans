import { Fragment, ReactNode, useCallback, useMemo, useState } from 'react';
import NiceModal from '@ebay/nice-modal-react';
import { PaletteMode } from '@mui/material';
import { AlertColor } from '@mui/material/Alert';
import Container from '@mui/material/Container';
import CssBaseline from '@mui/material/CssBaseline';
import { ThemeProvider } from '@mui/material/styles';
import useMediaQuery from '@mui/material/useMediaQuery';
import GlobalStyles from '@mui/system/GlobalStyles';
import { AdapterDateFns } from '@mui/x-date-pickers/AdapterDateFns';
import { LocalizationProvider } from '@mui/x-date-pickers/LocalizationProvider';
import { QueryClient } from '@tanstack/react-query';
import { createRootRouteWithContext, Outlet } from '@tanstack/react-router';
import { PermissionLevel } from '../api';
import { AuthContextProps } from '../auth.tsx';
import { Flash, Flashes } from '../component/Flashes.tsx';
import { Footer } from '../component/Footer.tsx';
import { LogoutHandler } from '../component/LogoutHandler.tsx';
import { NotificationsProvider } from '../component/NotificationsProvider.tsx';
import { QueueChat } from '../component/QueueChat.tsx';
import { QueueProvider } from '../component/QueueProvider.tsx';
import { TopBar } from '../component/TopBar.tsx';
import { ColourModeContext } from '../contexts/ColourModeContext.tsx';
import { UserFlashCtx } from '../contexts/UserFlashCtx.tsx';
import { useAuth } from '../hooks/useAuth.ts';
import * as bg from '../images/bg-orig.jpg';
import { createThemeByMode } from '../theme.ts';
import { checkFeatureEnabled } from '../util/features.ts';

type RouterContext = {
    auth: AuthContextProps;
    queryClient: QueryClient;
};

export const Route = createRootRouteWithContext<RouterContext>()({
    component: Root
});

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
        (level: AlertColor, message: string, heading = 'header', closable = true) => {
            if (flashes.length && flashes[flashes.length - 1]?.message == message) {
                // Skip duplicates
                return;
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

    return (
        <UserFlashCtx.Provider value={{ flashes, setFlashes, sendFlash }}>
            <QueueProvider>
                <LocalizationProvider dateAdapter={AdapterDateFns}>
                    <Fragment>
                        <ColourModeContext.Provider value={colorMode}>
                            <ThemeProvider theme={theme}>
                                <BackgroundImageProvider>
                                    <NotificationsProvider>
                                        <NiceModal.Provider>
                                            <LogoutHandler />
                                            <CssBaseline />
                                            <GlobalStyles
                                                styles={{
                                                    body: { backgroundImage: bg.default }
                                                }}
                                            />
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
                                </BackgroundImageProvider>
                            </ThemeProvider>
                        </ColourModeContext.Provider>
                    </Fragment>
                </LocalizationProvider>
            </QueueProvider>
        </UserFlashCtx.Provider>
    );
}

const BackgroundImageProvider = ({ children }: { children: ReactNode }) => {
    return (
        <div
            style={{
                backgroundColor: '#000',
                backgroundSize: '100%',
                backgroundImage: `url(${bg.default})`,
                backgroundRepeat: 'repeat',
                height: '100%'
            }}
        >
            {children}
        </div>
    );
};
