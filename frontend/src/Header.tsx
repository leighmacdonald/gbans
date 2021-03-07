import React from "react";
import {Link} from "react-router-dom"
import {PlayerProfile} from "./util/api";

interface HeaderProps {
    name: string
    profile: NonNullable<PlayerProfile>
    onLogin: () => void
    onLogout: () => void
}

export const Header = ({profile, onLogin, onLogout, name}: HeaderProps) => {
    console.log(`loading profile: ${profile}`)
    return (
        <header className="grid-container full">
            <nav className="grid-x" id="header_nav">
                <div className="cell">
                    <div className="top-bar">
                        <div className="top-bar-left">
                            <ul className="dropdown menu" data-dropdown-menu={true}>
                                <li className="menu-text">{name}</li>
                                <li>
                                    <Link to={`/`}><i className="fi-home"/> Home</Link>
                                    <ul className="menu vertical">
                                        <li><Link to="/servers">Servers</Link></li>
                                    </ul>
                                </li>
                                <li>
                                    <Link to={"/bans"}><i className="fi-torsos-all"/> Bans</Link>
                                    <ul className="menu vertical">
                                        <li><Link to="/bans">All Bans</Link></li>
                                        <li><Link to="/appeal">Appeal</Link></li>
                                    </ul>
                                </li>
                                <li>
                                    <Link to="/report">Report</Link>
                                </li>
                                <li>
                                    <Link to="/admin/ban">Ban Player</Link>
                                </li>
                            </ul>
                        </div>
                        <div className="top-bar-right">
                            <ul className="dropdown menu" data-dropdown-menu={true}>
                                {profile.player.steam_id === 0 && <>
                                    <li>
                                        <div className="btn_login" onClick={onLogin}/>
                                    </li>
                                </>}
                                {profile.player.steam_id > 0 && <>
                                    <li>
                                        <Link to="/admin"><i className="fi-widget"/> Admin</Link>
                                        <ul className="menu vertical">
                                            <li><Link to="/admin/people">People</Link></li>
                                            <li><Link to="/admin/import">Import</Link></li>
                                            <li><Link to="/admin/filters">Filtered Words</Link></li>
                                            <li><Link to="/admin/servers">Servers</Link></li>
                                            <li><Link to="/admin/server_logs">Server Logs</Link></li>
                                        </ul>
                                    </li>
                                    <li>
                                        <Link to="/profile"><img className="avatar" alt="Avatar"
                                                                 src={profile.player.avatarfull}/>
                                            <span>{profile.player.personaname}</span></Link>
                                        <ul className="menu vertical">
                                            <li><Link to="/settings">Settings</Link></li>
                                            <li><a className={"button"} onClick={onLogout}>Logout</a></li>
                                        </ul>
                                    </li>
                                </>
                                }
                            </ul>
                        </div>
                    </div>
                </div>
            </nav>
        </header>
    )
}