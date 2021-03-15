import React, { useEffect, useState } from 'react';
import { BrowserRouter as Router, Route, Switch } from 'react-router-dom';
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
import { apiGetCurrentProfile, PlayerProfile } from './util/api';
import { AdminBan } from './page/AdminBan';
import { AdminServerLog } from './page/AdminServerLog';
import { TopBar } from './component/TopBar';
import { makeStyles } from '@material-ui/core/styles';
import { Container } from '@material-ui/core';
import { UserFlashCtx } from './contexts/UserFlashCtx';
import { Logout } from './page/Logout';
import { Redirect } from 'react-router';
import { PageNotFound } from './page/PageNotFound';

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
    const [currentUser, setCurrentUser] = useState<NonNullable<PlayerProfile>>(
        GuestProfile
    );
    const [flashes, setFlashes] = useState<Flash[]>([]);

    useEffect(() => {
        const fetchProfile = async () => {
            const token = localStorage.getItem('token');
            if (token != null && token != '') {
                const profile = (await apiGetCurrentProfile()) as NonNullable<PlayerProfile>;
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
                        <Switch>
                            <Route exact path={'/'} component={Home} />
                            <Route
                                exact
                                path={'/servers'}
                                component={Servers}
                            />
                            <Route exact path={'/bans'} component={Bans} />
                            <Route exact path={'/appeal'} component={Appeal} />
                            <Route exact path={'/report'} component={Report} />
                            <Route
                                exact
                                path={'/settings'}
                                component={Settings}
                            />
                            <Route
                                path={'/profile/:steam_id'}
                                component={Profile}
                            />
                            <Route path={'/ban/:ban_id'} component={BanView} />
                            <Route
                                exact
                                path={'/admin/ban'}
                                component={AdminBan}
                            />
                            <Route
                                exact
                                path={'/admin/filters'}
                                component={AdminFilters}
                            />
                            <Route
                                exact
                                path={'/admin/reports'}
                                component={AdminReports}
                            />
                            <Route
                                exact
                                path={'/admin/import'}
                                component={AdminImport}
                            />
                            <Route
                                exact
                                path={'/admin/people'}
                                component={AdminPeople}
                            />
                            <Route
                                exact
                                path={'/admin/server_logs'}
                                component={AdminServerLog}
                            />
                            <Route
                                exact
                                path={'/admin/servers'}
                                component={AdminServers}
                            />
                            <Route
                                exact
                                path={'/login/success'}
                                component={LoginSuccess}
                            />
                            <Route exact path={'/logout'} component={Logout} />
                            <Route path="/404" component={PageNotFound} />
                            <Redirect to="/404" />
                        </Switch>
                        <Footer />
                    </main>
                </Container>
            </Router>
        </CurrentUserCtx.Provider>
    );
};
