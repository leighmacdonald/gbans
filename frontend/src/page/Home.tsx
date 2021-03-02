import React from "react";
import {StatsPanel} from "../component/StatsPanel";
import {BanList} from "../component/BanList";

export const Home = () => {
    return (
        <div className="grid-container">
            <div className="grid-x grid-padding-x">
                <div className="cell medium-9">
                    <h1>Recent Bans</h1>
                    <BanList/>
                </div>
                <div className="cell medium-3">
                    <h1>Stats</h1>
                    <StatsPanel />
                </div>
            </div>
        </div>
    )
}