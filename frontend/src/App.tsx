import React, {useContext, useEffect} from "react";
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
import {Auth, AuthContext} from "./contexts/Auth";
import {BanView} from "./page/BanView";
import {apiGetCurrentProfile, handleOnLogin, handleOnLogout, Person} from "./util/api";
import {Nullable} from "./util/types";
import {AdminBan} from "./page/AdminBan";

export const App = () => {
    const [currentProfile, setCurrentProfile] = React.useState<Nullable<Person>>()
    // @ts-ignore
    const [flashes, setFlashes] = React.useState<Flash[]>([])

    useEffect(() => {
        if (currentProfile != null && currentProfile.steam_id > 0) {
            const fetchProfile = async () => {
                const profile = await apiGetCurrentProfile() as Person;
                setCurrentProfile(profile)
            }
            // noinspection JSIgnoredPromiseFromCall
            fetchProfile();
        }
    }, [])

    const auth = useContext<AuthContext>(Auth);
    return (
        <Auth.Provider value={auth}>
            <Router>
                <Header name={"gbans"}
                        profile={currentProfile}
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
                    <Route path={"/admin/servers"} component={AdminServers}/>
                    <Route path={"/login/success"} component={LoginSuccess}/>
                </Switch>
                <Footer/>
            </Router>
        </Auth.Provider>
    )
}
