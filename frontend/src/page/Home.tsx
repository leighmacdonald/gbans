import React from "react";
import {StatsPanel, Stats} from "../component/StatsPanel";
import {BanList} from "../component/BanList";


export const Home = () => {
    const stats: Stats = {
        bans: 10,
        network_bans: 100,
        network_bans_hosts: 100000
    }
    return (
        <div className="grid-container">
            <div className="grid-x grid-padding-x">
                <div className="cell medium-10">
                    <h1>Recent Bans</h1>
                    <BanList/>
                </div>
                <div className="cell medium-2">
                    <h1>Stats</h1>
                    <StatsPanel {...stats} />
                </div>
            </div>
        </div>
    )
}