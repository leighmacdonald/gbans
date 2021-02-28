import React from "react";

export const Home = () => {
    return (
        <div className="grid-container">
            <div className="grid-x grid-padding-x">
                <div className="cell medium-10">
                    <h1>Recent Bans</h1>
                    <div id="ban_list"/>
                </div>
                <div className="cell medium-2">
                    <h1>Stats</h1>
                    <div id="stats"/>
                </div>
            </div>
        </div>
    )
}