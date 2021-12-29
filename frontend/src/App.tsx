import React, { useEffect, useState } from 'react';
import { BrowserRouter as Router, Route, Routes } from 'react-router-dom';
import { Home } from './page/Home';
import { Settings } from './page/Settings';
import { Appeal } from './page/Appeal';
import { Report } from './page/Report';
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
import {
    apiGetCurrentProfile,
    PermissionLevel,
    PlayerProfile
} from './util/api';
import { AdminBan } from './page/AdminBan';
import { AdminServerLog } from './page/AdminServerLog';
import { TopBar } from './component/TopBar';
import { makeStyles } from '@material-ui/core/styles';
import { Container } from '@material-ui/core';
import { UserFlashCtx } from './contexts/UserFlashCtx';
import { Logout } from './page/Logout';
import { PageNotFound } from './page/PageNotFound';
import { PrivateRoute } from './component/PrivateRoute';

const useStyles = makeStyles((theme) => ({
    toolbar: {
        display: 'flex',
        alignItems: 'center',
        justifyContent: 'flex-end',
        padding: theme.spacing(0, 1),
        // necessary for content to be below app bar
        ...theme.mixins.toolbar
    },
    content: {
        flexGrow: 1,
        padding: theme.spacing(3)
    }
}));

export const App = (): JSX.Element => {
    const classes = useStyles();
    const [currentUser, setCurrentUser] =
        useState<NonNullable<PlayerProfile>>(GuestProfile);
    const [flashes, setFlashes] = useState<Flash[]>([]);

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

    return (
        <CurrentUserCtx.Provider value={{ currentUser, setCurrentUser }}>
            <Router>
                <Container maxWidth={'lg'}>
                    <main className={classes.content}>
                        <div className={classes.toolbar} />
                        <TopBar />
                        <UserFlashCtx.Provider value={{ flashes, setFlashes }}>
                            <Flashes flashes={flashes} />
                        </UserFlashCtx.Provider>
                        <Routes>
                            <Route path={'/'}>
                                <Home />
                            </Route>
                            <Route path={'/servers'}>
                                <Servers />
                            </Route>
                            <Route path={'/bans'}>
                                <Bans />
                            </Route>
                            <Route path={'/appeal'}>
                                <Appeal />
                            </Route>
                            <Route path={'/report'}>
                                <Report />
                            </Route>
                            <Route path={'/settings'}>
                                <Settings />
                            </Route>
                            <Route path={'/profile/:steam_id'}>
                                <Profile />
                            </Route>
                            <Route path={'/ban/:ban_id'}>
                                <BanView />
                            </Route>
                            <Route path={'/admin/ban'}>
                                <AdminBan />
                            </Route>
                            <Route path={'/admin/filters'}>
                                <AdminFilters />
                            </Route>
                            <Route path={'/admin/reports'}>
                                <AdminReports />
                            </Route>
                            <PrivateRoute
                                permission={PermissionLevel.Admin}
                                path={'/admin/import'}
                            >
                                <AdminImport />
                            </PrivateRoute>

                            <Route path={'/admin/people'}>
                                <AdminPeople />
                            </Route>
                            <Route path={'/admin/server_logs'}>
                                <AdminServerLog />
                            </Route>
                            <Route path={'/admin/servers'}>
                                <AdminServers />
                            </Route>
                            <Route path={'/login/success'}>
                                <LoginSuccess />
                            </Route>
                            <Route path={'/logout'}>
                                <Logout />
                            </Route>
                            <Route path="/404">
                                <PageNotFound />
                            </Route>
                        </Routes>
                        <Footer />
                    </main>
                </Container>
            </Router>
        </CurrentUserCtx.Provider>
    );
};
