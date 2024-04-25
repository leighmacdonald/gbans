import { Fragment, StrictMode, useCallback, useMemo, useState } from 'react';
import NiceModal from '@ebay/nice-modal-react';
import { PaletteMode } from '@mui/material';
import { AlertColor } from '@mui/material/Alert';
import Container from '@mui/material/Container';
import CssBaseline from '@mui/material/CssBaseline';
import { ThemeProvider } from '@mui/material/styles';
import useMediaQuery from '@mui/material/useMediaQuery';
import { AdapterDateFns } from '@mui/x-date-pickers/AdapterDateFns';
import { LocalizationProvider } from '@mui/x-date-pickers/LocalizationProvider';
import { createRootRoute, Outlet } from '@tanstack/react-router';
import { TanStackRouterDevtools } from '@tanstack/router-devtools';
import { UserProfile } from '../api';
import { Flash, Flashes } from '../component/Flashes.tsx';
import { Footer } from '../component/Footer.tsx';
import { LogoutHandler } from '../component/LogoutHandler.tsx';
import { TopBar } from '../component/TopBar.tsx';
import { UserInit } from '../component/UserInit.tsx';
import { ColourModeContext } from '../contexts/ColourModeContext.tsx';
import { CurrentUserCtx } from '../contexts/CurrentUserCtx.tsx';
import { NotificationsProvider } from '../contexts/NotificationsCtx.tsx';
import { UserFlashCtx } from '../contexts/UserFlashCtx.tsx';
import { createThemeByMode } from '../theme.ts';
import { GuestProfile } from '../util/profile.ts';

export const Route = createRootRoute({
    component: Root
});

function Root() {
    const initialTheme =
        (localStorage.getItem('theme') as PaletteMode) || 'light';
    const [currentUser, setCurrentUser] =
        useState<NonNullable<UserProfile>>(GuestProfile);
    const [flashes, setFlashes] = useState<Flash[]>([]);

    const saveUser = (profile: UserProfile) => {
        setCurrentUser(profile);
    };

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
        (
            level: AlertColor,
            message: string,
            heading = 'header',
            closable = true
        ) => {
            if (
                flashes.length &&
                flashes[flashes.length - 1]?.message == message
            ) {
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
        <CurrentUserCtx.Provider
            value={{
                currentUser,
                setCurrentUser: saveUser
            }}
        >
            <UserFlashCtx.Provider value={{ flashes, setFlashes, sendFlash }}>
                <LocalizationProvider dateAdapter={AdapterDateFns}>
                    <Fragment>
                        <ColourModeContext.Provider value={colorMode}>
                            <ThemeProvider theme={theme}>
                                <NotificationsProvider>
                                    <StrictMode>
                                        <NiceModal.Provider>
                                            <UserInit />
                                            <LogoutHandler />
                                            <CssBaseline />
                                            <Container maxWidth={'lg'}>
                                                <TopBar />
                                                <div
                                                    style={{
                                                        marginTop: 24
                                                    }}
                                                >
                                                    <Outlet />
                                                    <TanStackRouterDevtools />
                                                </div>
                                                <Footer />
                                            </Container>
                                            <Flashes />
                                        </NiceModal.Provider>
                                    </StrictMode>
                                </NotificationsProvider>
                            </ThemeProvider>
                        </ColourModeContext.Provider>
                    </Fragment>
                </LocalizationProvider>
            </UserFlashCtx.Provider>
        </CurrentUserCtx.Provider>
    );
}
