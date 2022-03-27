import React, { useEffect, useState } from 'react';
import AdapterDateFns from '@mui/lab/AdapterDateFns';
import LocalizationProvider from '@mui/lab/LocalizationProvider';
import Container from '@mui/material/Container';
import CssBaseline from '@mui/material/CssBaseline';
import ThemeProvider from '@mui/material/styles/ThemeProvider';
import { BrowserRouter as Router, Route, Routes } from 'react-router-dom';
import { Home } from './page/Home';
import { Settings } from './page/Settings';
import { Appeal } from './page/Appeal';
import { ReportCreatePage } from './page/ReportCreatePage';
import { AdminReports } from './page/AdminReports';
import { AdminFilters } from './page/AdminFilters';
import { AdminImport } from './page/AdminImport';
import { AdminPeople } from './page/AdminPeople';
import { Bans } from './page/Bans';
import { Servers } from './page/Servers';
import { AdminServers } from './page/AdminServers';
import { Flash, Flashes } from './component/Flashes';
import { LoginSuccess } from './page/LoginSuccess';
import { Profile } from './page/Profile';
import { Footer } from './component/Footer';
import { CurrentUserCtx, GuestProfile } from './contexts/CurrentUserCtx';
import { BanView } from './page/BanView';
import { apiGetCurrentProfile, PermissionLevel, PlayerProfile } from './api';
import { AdminBan } from './page/AdminBan';
import { AdminServerLog } from './page/AdminServerLog';
import { TopBar } from './component/TopBar';
import { UserFlashCtx } from './contexts/UserFlashCtx';
import { Logout } from './page/Logout';
import { PageNotFound } from './page/PageNotFound';
import { PrivateRoute } from './component/PrivateRoute';
import { createThemeByMode } from './theme';
import { ReportViewPage } from './page/ReportViewPage';
import { PaletteMode, useMediaQuery } from '@mui/material';
import { ColourModeContext } from './contexts/ColourModeContext';

export const App = (): JSX.Element => {
    const [currentUser, setCurrentUser] =
        useState<NonNullable<PlayerProfile>>(GuestProfile);
    const [flashes, setFlashes] = useState<Flash[]>([]);

    let currentTheme = localStorage.getItem('theme') as PaletteMode;
    const prefersDarkMode = useMediaQuery('(prefers-color-scheme: dark)');
    if (!currentTheme) {
        currentTheme = prefersDarkMode ? 'dark' : 'light';
    }
    const [mode, setMode] = React.useState<'light' | 'dark'>(currentTheme);

    const updateMode = (prevMode: PaletteMode): PaletteMode => {
        const m = prevMode === 'light' ? 'dark' : ('light' as PaletteMode);
        localStorage.setItem('theme', m);
        return m;
    };

    const colorMode = React.useMemo(
        () => ({
            toggleColorMode: () => {
                setMode(updateMode);
            }
        }),
        []
    );

    useEffect(() => {
        const fetchProfile = async () => {
            const token = localStorage.getItem('token');
            if (token != null && token != '') {
                const profile =
                    (await apiGetCurrentProfile()) as NonNullable<PlayerProfile>;
                setCurrentUser(profile);
            }
        };
        // noinspection JSIgnoredPromiseFromCall
        fetchProfile();
    }, [setCurrentUser]);

    const theme = React.useMemo(() => createThemeByMode(mode), [mode]);

    return (
        <CurrentUserCtx.Provider value={{ currentUser, setCurrentUser }}>
            <UserFlashCtx.Provider value={{ flashes, setFlashes }}>
                <LocalizationProvider dateAdapter={AdapterDateFns}>
                    <Router>
                        <React.Fragment>
                            <ColourModeContext.Provider value={colorMode}>
                                <ThemeProvider theme={theme}>
                                    <React.StrictMode>
                                        <CssBaseline />
                                        <Container maxWidth={'lg'}>
                                            <TopBar />

                                            <Flashes />

                                            <Routes>
                                                <Route
                                                    path={'/'}
                                                    element={<Home />}
                                                />
                                                <Route
                                                    path={'/servers'}
                                                    element={<Servers />}
                                                />
                                                <Route
                                                    path={'/bans'}
                                                    element={<Bans />}
                                                />
                                                <Route
                                                    path={'/appeal'}
                                                    element={<Appeal />}
                                                />
                                                <Route
                                                    path={'/report/:report_id'}
                                                    element={<ReportViewPage />}
                                                />
                                                <Route
                                                    path={'/report'}
                                                    element={
                                                        <PrivateRoute
                                                            permission={
                                                                PermissionLevel.Authenticated
                                                            }
                                                        >
                                                            <ReportCreatePage />
                                                        </PrivateRoute>
                                                    }
                                                />
                                                <Route
                                                    path={'/settings'}
                                                    element={<Settings />}
                                                />
                                                <Route
                                                    path={'/profile/:steam_id'}
                                                    element={<Profile />}
                                                />
                                                <Route
                                                    path={'/ban/:ban_id'}
                                                    element={<BanView />}
                                                />
                                                <Route
                                                    path={'/admin/ban'}
                                                    element={<AdminBan />}
                                                />
                                                <Route
                                                    path={'/admin/filters'}
                                                    element={<AdminFilters />}
                                                />
                                                <Route
                                                    path={'/admin/reports'}
                                                    element={<AdminReports />}
                                                />
                                                <Route
                                                    path={'/admin/import'}
                                                    element={
                                                        <PrivateRoute
                                                            permission={
                                                                PermissionLevel.Admin
                                                            }
                                                        >
                                                            <AdminImport />
                                                        </PrivateRoute>
                                                    }
                                                />

                                                <Route
                                                    path={'/admin/people'}
                                                    element={<AdminPeople />}
                                                />
                                                <Route
                                                    path={'/admin/server_logs'}
                                                    element={<AdminServerLog />}
                                                />
                                                <Route
                                                    path={'/admin/servers'}
                                                    element={<AdminServers />}
                                                />
                                                <Route
                                                    path={'/login/success'}
                                                    element={<LoginSuccess />}
                                                />
                                                <Route
                                                    path={'/logout'}
                                                    element={<Logout />}
                                                />
                                                <Route
                                                    path="/404"
                                                    element={<PageNotFound />}
                                                />
                                            </Routes>
                                            <Footer />
                                        </Container>
                                    </React.StrictMode>
                                </ThemeProvider>
                            </ColourModeContext.Provider>
                        </React.Fragment>
                    </Router>
                </LocalizationProvider>
            </UserFlashCtx.Provider>
        </CurrentUserCtx.Provider>
    );
};
