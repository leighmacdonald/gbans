import React, { useCallback, useEffect, useMemo, useState } from 'react';
import Container from '@mui/material/Container';
import CssBaseline from '@mui/material/CssBaseline';
import ThemeProvider from '@mui/material/styles/ThemeProvider';
import { BrowserRouter as Router, Route, Routes } from 'react-router-dom';
import { Home } from './page/Home';
import { Settings } from './page/Settings';
import { ReportCreatePage } from './page/ReportCreatePage';
import { AdminReports } from './page/AdminReports';
import { AdminFilters } from './page/AdminFilters';
import { AdminImport } from './page/AdminImport';
import { AdminPeople } from './page/AdminPeople';
import { Bans } from './page/Bans';
import { Servers } from './page/Servers';
import { AdminServers } from './page/AdminServers';
import { Flash } from './component/Flashes';
import { LoginSuccess } from './page/LoginSuccess';
import { Profile } from './page/Profile';
import { Footer } from './component/Footer';
import { CurrentUserCtx, GuestProfile } from './contexts/CurrentUserCtx';
import { BanPage } from './page/BanPage';
import { apiGetCurrentProfile, PermissionLevel, UserProfile } from './api';
import { AdminBan } from './page/AdminBan';
import { TopBar } from './component/TopBar';
import { UserFlashCtx } from './contexts/UserFlashCtx';
import { Logout } from './page/Logout';
import { PageNotFound } from './page/PageNotFound';
import { PrivateRoute } from './component/PrivateRoute';
import { createThemeByMode } from './theme';
import { ReportViewPage } from './page/ReportViewPage';
import { PaletteMode } from '@mui/material';
import useMediaQuery from '@mui/material/useMediaQuery';
import { ColourModeContext } from './contexts/ColourModeContext';
import { AdminNews } from './page/AdminNews';
import { WikiPage } from './page/WikiPage';
import { logErr } from './util/errors';
import { MatchPage } from './page/MatchPage';
import { AlertColor } from '@mui/material/Alert/Alert';
import { MatchListPage } from './page/MatchListPage';
import { LocalizationProvider } from '@mui/x-date-pickers/LocalizationProvider';
import { AdapterDateFns } from '@mui/x-date-pickers/AdapterDateFns';

export const App = (): JSX.Element => {
    const [currentUser, setCurrentUser] =
        useState<NonNullable<UserProfile>>(GuestProfile);
    const [flashes, setFlashes] = useState<Flash[]>([]);

    let currentTheme = localStorage.getItem('theme') as PaletteMode;
    const prefersDarkMode = useMediaQuery('(prefers-color-scheme: dark)');
    if (!currentTheme) {
        currentTheme = prefersDarkMode ? 'dark' : 'light';
    }
    const [mode, setMode] = useState<'light' | 'dark'>(currentTheme);

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
    //NonNullable<UserProfile>
    useEffect(() => {
        const token = localStorage.getItem('token');
        if (token != null && token != '') {
            apiGetCurrentProfile()
                .then((profile) => {
                    setCurrentUser(profile);
                })
                .catch(logErr);
        }
    }, [setCurrentUser]);

    const theme = useMemo(() => createThemeByMode(mode), [mode]);

    const sendFlash = useCallback(
        (
            level: AlertColor,
            message: string,
            heading = 'header',
            closable = true
        ) => {
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
        <CurrentUserCtx.Provider value={{ currentUser, setCurrentUser }}>
            <UserFlashCtx.Provider value={{ flashes, setFlashes, sendFlash }}>
                <LocalizationProvider dateAdapter={AdapterDateFns}>
                    <Router>
                        <React.Fragment>
                            <ColourModeContext.Provider value={colorMode}>
                                <ThemeProvider theme={theme}>
                                    <React.StrictMode>
                                        <CssBaseline />
                                        <Container maxWidth={'lg'}>
                                            <TopBar />

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
                                                    path={'/wiki'}
                                                    element={<WikiPage />}
                                                />
                                                <Route
                                                    path={'/wiki/:slug'}
                                                    element={<WikiPage />}
                                                />
                                                <Route
                                                    path={'/ban/:ban_id'}
                                                    element={<BanPage />}
                                                />
                                                <Route
                                                    path={'/report/:report_id'}
                                                    element={<ReportViewPage />}
                                                />
                                                <Route
                                                    path={'/log/:match_id'}
                                                    element={<MatchPage />}
                                                />
                                                <Route
                                                    path={'/logs'}
                                                    element={<MatchListPage />}
                                                />
                                                <Route
                                                    path={'/report'}
                                                    element={
                                                        <PrivateRoute
                                                            permission={
                                                                PermissionLevel.User
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
                                                    element={<BanPage />}
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
                                                    path={'/admin/news'}
                                                    element={<AdminNews />}
                                                />
                                                <Route
                                                    path={'/admin/people'}
                                                    element={<AdminPeople />}
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
