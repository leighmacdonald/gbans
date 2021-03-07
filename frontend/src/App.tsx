import React, {useEffect, useState} from "react";
import {BrowserRouter as Router, Route, Switch} from "react-router-dom";
import {Home} from "./page/Home";
import {Settings} from "./page/Settings";
import {Appeal} from "./page/Appeal";
import {Report} from "./page/Report";
import {AdminReports} from "./page/AdminReports";
import {AdminFilters} from "./page/AdminFilters";
import {AdminImport} from "./page/AdminImport";
import {AdminPeople} from "./page/AdminPeople";
import {Bans} from "./page/Bans";
import {Servers} from "./page/Servers";
import {AdminServers} from "./page/AdminServers";
import {Header} from "./Header";
import {Flash, Flashes} from "./component/Flashes";
import {LoginSuccess} from "./page/LoginSuccess";
import {Profile} from "./page/Profile";
import {Footer} from "./component/Footer";
import {CurrentUserCtx, GuestProfile} from "./contexts/CurrentUserCtx";
import {BanView} from "./page/BanView";
import {
    apiGetCurrentProfile,
    handleOnLogin,
    handleOnLogout,
    PlayerProfile
} from "./util/api";
import {AdminBan} from "./page/AdminBan";
import {AdminServerLog} from "./page/AdminServerLog";

export const App = () => {
    const [currentUser, setCurrentUser] = useState<NonNullable<PlayerProfile>>(GuestProfile)
    // @ts-ignore
    const [flashes, setFlashes] = React.useState<Flash[]>([])

    useEffect(() => {
        const fetchProfile = async () => {
            const token = localStorage.getItem("token")
            if (token != null && token != "") {
                const profile = await apiGetCurrentProfile() as NonNullable<PlayerProfile>;
                setCurrentUser(profile)
            }
        }
        // noinspection JSIgnoredPromiseFromCall
        fetchProfile();
    }, [setCurrentUser])

    return (
        <CurrentUserCtx.Provider value={{currentUser, setCurrentUser}}>
            <Router>
                <Header name={"gbans"}
                        profile={currentUser}
                        onLogin={handleOnLogin}
                        onLogout={handleOnLogout}/>
                <Flashes flashes={flashes}/>
                <Switch>
                    <Route path={"/"} component={Home} exact={true}/>
                    <Route path={"/servers"} component={Servers}/>
                    <Route path={"/bans"} component={Bans}/>
                    <Route path={"/appeal"} component={Appeal}/>
                    <Route path={"/report"} component={Report}/>
                    <Route path={"/settings"} component={Settings}/>
                    <Route path={"/profile"} component={Profile}/>
                    <Route path={"/ban/:ban_id"} component={BanView}/>
                    <Route path={"/admin/ban"} component={AdminBan}/>
                    <Route path={"/admin/filters"} component={AdminFilters}/>
                    <Route path={"/admin/reports"} component={AdminReports}/>
                    <Route path={"/admin/import"} component={AdminImport}/>
                    <Route path={"/admin/people"} component={AdminPeople}/>
                    <Route path={"/admin/server_logs"} component={AdminServerLog}/>
                    <Route path={"/admin/servers"} component={AdminServers}/>
                    <Route path={"/login/success"} component={LoginSuccess}/>
                </Switch>
                <Footer/>
            </Router>
        </CurrentUserCtx.Provider>
    )
}
