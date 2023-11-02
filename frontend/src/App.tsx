import React, { useCallback, useMemo, useState, JSX } from 'react';
import { BrowserRouter as Router, Route, Routes } from 'react-router-dom';
import NiceModal from '@ebay/nice-modal-react';
import { PaletteMode } from '@mui/material';
import { AlertColor } from '@mui/material/Alert';
import Container from '@mui/material/Container';
import CssBaseline from '@mui/material/CssBaseline';
import { ThemeProvider } from '@mui/material/styles';
import useMediaQuery from '@mui/material/useMediaQuery';
import { AdapterDateFns } from '@mui/x-date-pickers/AdapterDateFns';
import { LocalizationProvider } from '@mui/x-date-pickers/LocalizationProvider';
import { PermissionLevel, UserProfile } from './api';
import { ErrorBoundary } from './component/ErrorBoundary';
import { Flash } from './component/Flashes';
import { Footer } from './component/Footer';
import { LogoutHandler } from './component/LogoutHandler';
import { PrivateRoute } from './component/PrivateRoute';
import { TopBar } from './component/TopBar';
import { UserInit } from './component/UserInit';
import { ColourModeContext } from './contexts/ColourModeContext';
import { CurrentUserCtx, GuestProfile } from './contexts/CurrentUserCtx';
import { NotificationsProvider } from './contexts/NotificationsCtx';
import { UserFlashCtx } from './contexts/UserFlashCtx';
import { AdminAppeals } from './page/AdminAppeals';
import { AdminBan } from './page/AdminBan';
import { AdminContests } from './page/AdminContests';
import { AdminFilters } from './page/AdminFilters';
import { AdminImport } from './page/AdminImport';
import { AdminNetworkPage } from './page/AdminNetworkPage';
import { AdminNews } from './page/AdminNews';
import { AdminPeople } from './page/AdminPeople';
import { AdminReports } from './page/AdminReports';
import { AdminServers } from './page/AdminServers';
import { BanPage } from './page/BanPage';
import { ChatLogPage } from './page/ChatLogPage';
import { ContestListPage } from './page/ContestListPage';
import { ContestPage } from './page/ContestPage';
import { Home } from './page/Home';
import { Login } from './page/Login';
import { LoginDiscordSuccess } from './page/LoginDiscordSuccess';
import { LoginSteamSuccess } from './page/LoginSteamSuccess';
import { Logout } from './page/Logout';
import { MatchListPage } from './page/MatchListPage';
import { MatchPage } from './page/MatchPage';
import { NotificationsPage } from './page/NotificationsPage';
import { PageNotFound } from './page/PageNotFound';
import { PlayerStatsPage } from './page/PlayerStatsPage';
import { PrivacyPolicyPage } from './page/PrivacyPolicyPage';
import { Profile } from './page/Profile';
import { ProfileSettingsPage } from './page/ProfileSettingsPage';
import { ReportCreatePage } from './page/ReportCreatePage';
import { ReportViewPage } from './page/ReportViewPage';
import { STVPage } from './page/STVPage';
import { Servers } from './page/Servers';
import { StatsPage } from './page/StatsPage';
import { StatsWeaponOverallPage } from './page/StatsWeaponOverallPage';
import { WikiPage } from './page/WikiPage';
import { createThemeByMode } from './theme';

export interface AppProps {
    initialTheme: PaletteMode;
}

export const App = ({ initialTheme }: AppProps): JSX.Element => {
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
                    <Router>
                        <React.Fragment>
                            <ColourModeContext.Provider value={colorMode}>
                                <ThemeProvider theme={theme}>
                                    <NotificationsProvider>
                                        <React.StrictMode>
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
                                                        <ErrorBoundary>
                                                            <Routes>
                                                                <Route
                                                                    path={'/'}
                                                                    element={
                                                                        <ErrorBoundary>
                                                                            <Home />
                                                                        </ErrorBoundary>
                                                                    }
                                                                />
                                                                <Route
                                                                    path={
                                                                        '/servers'
                                                                    }
                                                                    element={
                                                                        <ErrorBoundary>
                                                                            <Servers />
                                                                        </ErrorBoundary>
                                                                    }
                                                                />
                                                                <Route
                                                                    path={
                                                                        '/stv'
                                                                    }
                                                                    element={
                                                                        <ErrorBoundary>
                                                                            <STVPage />
                                                                        </ErrorBoundary>
                                                                    }
                                                                />
                                                                <Route
                                                                    path={
                                                                        '/login/success'
                                                                    }
                                                                    element={
                                                                        <ErrorBoundary>
                                                                            <LoginSteamSuccess />
                                                                        </ErrorBoundary>
                                                                    }
                                                                />
                                                                <Route
                                                                    path={
                                                                        '/privacy-policy'
                                                                    }
                                                                    element={
                                                                        <ErrorBoundary>
                                                                            <PrivacyPolicyPage />
                                                                        </ErrorBoundary>
                                                                    }
                                                                />
                                                                <Route
                                                                    path={
                                                                        '/contests'
                                                                    }
                                                                    element={
                                                                        <ErrorBoundary>
                                                                            <ContestListPage />
                                                                        </ErrorBoundary>
                                                                    }
                                                                />
                                                                <Route
                                                                    path={'/'}
                                                                    element={
                                                                        <ErrorBoundary>
                                                                            <Home />
                                                                        </ErrorBoundary>
                                                                    }
                                                                />

                                                                <Route
                                                                    path={
                                                                        '/servers'
                                                                    }
                                                                    element={
                                                                        <ErrorBoundary>
                                                                            <Servers />
                                                                        </ErrorBoundary>
                                                                    }
                                                                />

                                                                <Route
                                                                    path={
                                                                        '/stv'
                                                                    }
                                                                    element={
                                                                        <ErrorBoundary>
                                                                            <STVPage />
                                                                        </ErrorBoundary>
                                                                    }
                                                                />
                                                                <Route
                                                                    path={
                                                                        '/login/success'
                                                                    }
                                                                    element={
                                                                        <ErrorBoundary>
                                                                            <LoginSteamSuccess />
                                                                        </ErrorBoundary>
                                                                    }
                                                                />
                                                                <Route
                                                                    path={
                                                                        '/contests'
                                                                    }
                                                                    element={
                                                                        <ErrorBoundary>
                                                                            <ContestListPage />
                                                                        </ErrorBoundary>
                                                                    }
                                                                />

                                                                <Route
                                                                    path={
                                                                        '/contests/:contest_id'
                                                                    }
                                                                    element={
                                                                        <ErrorBoundary>
                                                                            <ContestPage />
                                                                        </ErrorBoundary>
                                                                    }
                                                                />

                                                                <Route
                                                                    path={
                                                                        '/stats'
                                                                    }
                                                                    element={
                                                                        <ErrorBoundary>
                                                                            <PrivateRoute
                                                                                permission={
                                                                                    PermissionLevel.User
                                                                                }
                                                                            >
                                                                                <StatsPage />
                                                                            </PrivateRoute>
                                                                        </ErrorBoundary>
                                                                    }
                                                                />
                                                                <Route
                                                                    path={
                                                                        '/stats/player/:steam_id'
                                                                    }
                                                                    element={
                                                                        <ErrorBoundary>
                                                                            <PrivateRoute
                                                                                permission={
                                                                                    PermissionLevel.User
                                                                                }
                                                                            >
                                                                                <PlayerStatsPage />
                                                                            </PrivateRoute>
                                                                        </ErrorBoundary>
                                                                    }
                                                                />
                                                                <Route
                                                                    path={
                                                                        '/stats/weapon/:weapon_id'
                                                                    }
                                                                    element={
                                                                        <ErrorBoundary>
                                                                            <PrivateRoute
                                                                                permission={
                                                                                    PermissionLevel.User
                                                                                }
                                                                            >
                                                                                <StatsWeaponOverallPage />
                                                                            </PrivateRoute>
                                                                        </ErrorBoundary>
                                                                    }
                                                                />

                                                                <Route
                                                                    path={
                                                                        '/wiki'
                                                                    }
                                                                    element={
                                                                        <ErrorBoundary>
                                                                            <WikiPage />
                                                                        </ErrorBoundary>
                                                                    }
                                                                />
                                                                <Route
                                                                    path={
                                                                        '/wiki/:slug'
                                                                    }
                                                                    element={
                                                                        <ErrorBoundary>
                                                                            <WikiPage />
                                                                        </ErrorBoundary>
                                                                    }
                                                                />
                                                                <Route
                                                                    path={
                                                                        '/ban/:ban_id'
                                                                    }
                                                                    element={
                                                                        <ErrorBoundary>
                                                                            <BanPage />
                                                                        </ErrorBoundary>
                                                                    }
                                                                />
                                                                <Route
                                                                    path={
                                                                        '/report/:report_id'
                                                                    }
                                                                    element={
                                                                        <ErrorBoundary>
                                                                            <ReportViewPage />
                                                                        </ErrorBoundary>
                                                                    }
                                                                />
                                                                <Route
                                                                    path={
                                                                        '/log/:match_id'
                                                                    }
                                                                    element={
                                                                        <ErrorBoundary>
                                                                            <PrivateRoute
                                                                                permission={
                                                                                    PermissionLevel.User
                                                                                }
                                                                            >
                                                                                <MatchPage />
                                                                            </PrivateRoute>
                                                                        </ErrorBoundary>
                                                                    }
                                                                />
                                                                <Route
                                                                    path={
                                                                        '/logs/:steam_id'
                                                                    }
                                                                    element={
                                                                        <ErrorBoundary>
                                                                            <PrivateRoute
                                                                                permission={
                                                                                    PermissionLevel.User
                                                                                }
                                                                            >
                                                                                <MatchListPage />
                                                                            </PrivateRoute>
                                                                        </ErrorBoundary>
                                                                    }
                                                                />
                                                                <Route
                                                                    path={
                                                                        '/report'
                                                                    }
                                                                    element={
                                                                        <ErrorBoundary>
                                                                            <PrivateRoute
                                                                                permission={
                                                                                    PermissionLevel.User
                                                                                }
                                                                            >
                                                                                <ReportCreatePage />
                                                                            </PrivateRoute>
                                                                        </ErrorBoundary>
                                                                    }
                                                                />
                                                                <Route
                                                                    path={
                                                                        '/notifications'
                                                                    }
                                                                    element={
                                                                        <ErrorBoundary>
                                                                            <PrivateRoute
                                                                                permission={
                                                                                    PermissionLevel.User
                                                                                }
                                                                            >
                                                                                <NotificationsPage />
                                                                            </PrivateRoute>
                                                                        </ErrorBoundary>
                                                                    }
                                                                />
                                                                <Route
                                                                    path={
                                                                        '/settings'
                                                                    }
                                                                    element={
                                                                        <ErrorBoundary>
                                                                            <PrivateRoute
                                                                                permission={
                                                                                    PermissionLevel.User
                                                                                }
                                                                            >
                                                                                <ProfileSettingsPage />
                                                                            </PrivateRoute>
                                                                        </ErrorBoundary>
                                                                    }
                                                                />
                                                                <Route
                                                                    path={
                                                                        '/profile/:steam_id'
                                                                    }
                                                                    element={
                                                                        <ErrorBoundary>
                                                                            <Profile />
                                                                        </ErrorBoundary>
                                                                    }
                                                                />

                                                                <Route
                                                                    path={
                                                                        '/report'
                                                                    }
                                                                    element={
                                                                        <ErrorBoundary>
                                                                            <PrivateRoute
                                                                                permission={
                                                                                    PermissionLevel.User
                                                                                }
                                                                            >
                                                                                <Route
                                                                                    path={
                                                                                        '/ban/:ban_id'
                                                                                    }
                                                                                    element={
                                                                                        <BanPage />
                                                                                    }
                                                                                />
                                                                            </PrivateRoute>
                                                                        </ErrorBoundary>
                                                                    }
                                                                />
                                                                <Route
                                                                    path={
                                                                        '/admin/ban'
                                                                    }
                                                                    element={
                                                                        <ErrorBoundary>
                                                                            <PrivateRoute
                                                                                permission={
                                                                                    PermissionLevel.Moderator
                                                                                }
                                                                            >
                                                                                <AdminBan />
                                                                            </PrivateRoute>
                                                                        </ErrorBoundary>
                                                                    }
                                                                />
                                                                <Route
                                                                    path={
                                                                        '/admin/filters'
                                                                    }
                                                                    element={
                                                                        <ErrorBoundary>
                                                                            <PrivateRoute
                                                                                permission={
                                                                                    PermissionLevel.Editor
                                                                                }
                                                                            >
                                                                                <AdminFilters />
                                                                            </PrivateRoute>
                                                                        </ErrorBoundary>
                                                                    }
                                                                />
                                                                <Route
                                                                    path={
                                                                        '/admin/network'
                                                                    }
                                                                    element={
                                                                        <ErrorBoundary>
                                                                            <PrivateRoute
                                                                                permission={
                                                                                    PermissionLevel.Editor
                                                                                }
                                                                            >
                                                                                <AdminNetworkPage />
                                                                            </PrivateRoute>
                                                                        </ErrorBoundary>
                                                                    }
                                                                />
                                                                <Route
                                                                    path={
                                                                        '/admin/reports'
                                                                    }
                                                                    element={
                                                                        <ErrorBoundary>
                                                                            <PrivateRoute
                                                                                permission={
                                                                                    PermissionLevel.Moderator
                                                                                }
                                                                            >
                                                                                <AdminReports />
                                                                            </PrivateRoute>
                                                                        </ErrorBoundary>
                                                                    }
                                                                />
                                                                <Route
                                                                    path={
                                                                        '/admin/contests'
                                                                    }
                                                                    element={
                                                                        <ErrorBoundary>
                                                                            <PrivateRoute
                                                                                permission={
                                                                                    PermissionLevel.Moderator
                                                                                }
                                                                            >
                                                                                <AdminContests />
                                                                            </PrivateRoute>
                                                                        </ErrorBoundary>
                                                                    }
                                                                />
                                                                <Route
                                                                    path={
                                                                        '/admin/appeals'
                                                                    }
                                                                    element={
                                                                        <ErrorBoundary>
                                                                            <PrivateRoute
                                                                                permission={
                                                                                    PermissionLevel.Moderator
                                                                                }
                                                                            >
                                                                                <AdminAppeals />
                                                                            </PrivateRoute>
                                                                        </ErrorBoundary>
                                                                    }
                                                                />
                                                                <Route
                                                                    path={
                                                                        '/admin/import'
                                                                    }
                                                                    element={
                                                                        <ErrorBoundary>
                                                                            <PrivateRoute
                                                                                permission={
                                                                                    PermissionLevel.Admin
                                                                                }
                                                                            >
                                                                                <AdminImport />
                                                                            </PrivateRoute>
                                                                        </ErrorBoundary>
                                                                    }
                                                                />
                                                                <Route
                                                                    path={
                                                                        '/admin/news'
                                                                    }
                                                                    element={
                                                                        <ErrorBoundary>
                                                                            <PrivateRoute
                                                                                permission={
                                                                                    PermissionLevel.Editor
                                                                                }
                                                                            >
                                                                                <AdminNews />
                                                                            </PrivateRoute>
                                                                        </ErrorBoundary>
                                                                    }
                                                                />
                                                                <Route
                                                                    path={
                                                                        '/chatlogs'
                                                                    }
                                                                    element={
                                                                        <ErrorBoundary>
                                                                            <PrivateRoute
                                                                                permission={
                                                                                    PermissionLevel.User
                                                                                }
                                                                            >
                                                                                <ChatLogPage />
                                                                            </PrivateRoute>
                                                                        </ErrorBoundary>
                                                                    }
                                                                />
                                                                <Route
                                                                    path={
                                                                        '/admin/people'
                                                                    }
                                                                    element={
                                                                        <ErrorBoundary>
                                                                            <PrivateRoute
                                                                                permission={
                                                                                    PermissionLevel.Admin
                                                                                }
                                                                            >
                                                                                <AdminPeople />
                                                                            </PrivateRoute>
                                                                        </ErrorBoundary>
                                                                    }
                                                                />
                                                                <Route
                                                                    path={
                                                                        '/admin/people'
                                                                    }
                                                                    element={
                                                                        <ErrorBoundary>
                                                                            <PrivateRoute
                                                                                permission={
                                                                                    PermissionLevel.Admin
                                                                                }
                                                                            >
                                                                                <AdminPeople />
                                                                            </PrivateRoute>
                                                                        </ErrorBoundary>
                                                                    }
                                                                />
                                                                <Route
                                                                    path={
                                                                        '/admin/servers'
                                                                    }
                                                                    element={
                                                                        <ErrorBoundary>
                                                                            <PrivateRoute
                                                                                permission={
                                                                                    PermissionLevel.Admin
                                                                                }
                                                                            >
                                                                                <AdminServers />
                                                                            </PrivateRoute>
                                                                        </ErrorBoundary>
                                                                    }
                                                                />
                                                                <Route
                                                                    path={
                                                                        '/login'
                                                                    }
                                                                    element={
                                                                        <ErrorBoundary>
                                                                            <Login />
                                                                        </ErrorBoundary>
                                                                    }
                                                                />
                                                                <Route
                                                                    path={
                                                                        '/login/discord'
                                                                    }
                                                                    element={
                                                                        <ErrorBoundary>
                                                                            <PrivateRoute
                                                                                permission={
                                                                                    PermissionLevel.User
                                                                                }
                                                                            >
                                                                                <LoginDiscordSuccess />
                                                                            </PrivateRoute>
                                                                        </ErrorBoundary>
                                                                    }
                                                                />
                                                                <Route
                                                                    path={
                                                                        '/logout'
                                                                    }
                                                                    element={
                                                                        <ErrorBoundary>
                                                                            <Logout />
                                                                        </ErrorBoundary>
                                                                    }
                                                                />
                                                                <Route
                                                                    path="/404"
                                                                    element={
                                                                        <ErrorBoundary>
                                                                            <PageNotFound />
                                                                        </ErrorBoundary>
                                                                    }
                                                                />
                                                            </Routes>
                                                        </ErrorBoundary>
                                                    </div>
                                                    <Footer />
                                                </Container>
                                            </NiceModal.Provider>
                                        </React.StrictMode>
                                    </NotificationsProvider>
                                </ThemeProvider>
                            </ColourModeContext.Provider>
                        </React.Fragment>
                    </Router>
                </LocalizationProvider>
            </UserFlashCtx.Provider>
        </CurrentUserCtx.Provider>
    );
};
