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
import { AdminAppealsPage } from './page/AdminAppealsPage';
import { AdminBanPage } from './page/AdminBanPage';
import { AdminContestsPage } from './page/AdminContestsPage';
import { AdminFiltersPage } from './page/AdminFiltersPage';
import { AdminImportPage } from './page/AdminImportPage';
import { AdminNetworkPage } from './page/AdminNetworkPage';
import { AdminNewsPage } from './page/AdminNewsPage';
import { AdminPeoplePage } from './page/AdminPeoplePage';
import { AdminReportsPage } from './page/AdminReportsPage';
import { AdminServersPage } from './page/AdminServersPage';
import { BanPage } from './page/BanPage';
import { ChatLogPage } from './page/ChatLogPage';
import { ContestListPage } from './page/ContestListPage';
import { ContestPage } from './page/ContestPage';
import { ForumOverviewPage } from './page/ForumOverviewPage';
import { ForumPage } from './page/ForumPage';
import { HomePage } from './page/HomePage';
import { LoginDiscordSuccessPage } from './page/LoginDiscordSuccessPage';
import { LoginPage } from './page/LoginPage';
import { LoginSteamSuccessPage } from './page/LoginSteamSuccessPage';
import { LogoutPage } from './page/LogoutPage';
import { MatchListPage } from './page/MatchListPage';
import { MatchPage } from './page/MatchPage';
import { NotificationsPage } from './page/NotificationsPage';
import { PageNotFoundPage } from './page/PageNotFoundPage';
import { PlayerStatsPage } from './page/PlayerStatsPage';
import { PrivacyPolicyPage } from './page/PrivacyPolicyPage';
import { ProfilePage } from './page/ProfilePage';
import { ProfileSettingsPage } from './page/ProfileSettingsPage';
import { ReportCreatePage } from './page/ReportCreatePage';
import { ReportViewPage } from './page/ReportViewPage';
import { STVPage } from './page/STVPage';
import { ServersPage } from './page/ServersPage';
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
                                                                            <HomePage />
                                                                        </ErrorBoundary>
                                                                    }
                                                                />
                                                                <Route
                                                                    path={
                                                                        '/servers'
                                                                    }
                                                                    element={
                                                                        <ErrorBoundary>
                                                                            <ServersPage />
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
                                                                            <LoginSteamSuccessPage />
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
                                                                            <HomePage />
                                                                        </ErrorBoundary>
                                                                    }
                                                                />

                                                                <Route
                                                                    path={
                                                                        '/servers'
                                                                    }
                                                                    element={
                                                                        <ErrorBoundary>
                                                                            <ServersPage />
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
                                                                            <LoginSteamSuccessPage />
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
                                                                        '/forums'
                                                                    }
                                                                    element={
                                                                        <ErrorBoundary>
                                                                            <PrivateRoute
                                                                                permission={
                                                                                    PermissionLevel.Moderator
                                                                                }
                                                                            >
                                                                                <ForumOverviewPage />
                                                                            </PrivateRoute>
                                                                        </ErrorBoundary>
                                                                    }
                                                                />

                                                                <Route
                                                                    path={
                                                                        '/forums/:forum_id'
                                                                    }
                                                                    element={
                                                                        <ErrorBoundary>
                                                                            <PrivateRoute
                                                                                permission={
                                                                                    PermissionLevel.Moderator
                                                                                }
                                                                            >
                                                                                <ForumPage />
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
                                                                            <ProfilePage />
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
                                                                                <AdminBanPage />
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
                                                                                <AdminFiltersPage />
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
                                                                                <AdminReportsPage />
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
                                                                                <AdminContestsPage />
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
                                                                                <AdminAppealsPage />
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
                                                                                <AdminImportPage />
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
                                                                                <AdminNewsPage />
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
                                                                                <AdminPeoplePage />
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
                                                                                <AdminPeoplePage />
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
                                                                                <AdminServersPage />
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
                                                                            <LoginPage />
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
                                                                                <LoginDiscordSuccessPage />
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
                                                                            <LogoutPage />
                                                                        </ErrorBoundary>
                                                                    }
                                                                />
                                                                <Route
                                                                    path="/404"
                                                                    element={
                                                                        <ErrorBoundary>
                                                                            <PageNotFoundPage />
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
