import React from "react";

export interface Stats {
    bans: number
    network_bans: number
    network_bans_hosts: number
}

export const StatsPanel = (stats: Stats) => {
    return (
        <>
            <div className={"grid-y"}>
                <div className={"cell"}>
                    <h2>Stats</h2>
                    <span>{stats.bans}</span>
                </div>
            </div>
        </>
    )
}