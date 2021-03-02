import React, {useEffect} from "react";
import {apiGetStats, DatabaseStats} from "../util/api";

export const StatsPanel = () => {
    const [stats, setStats] = React.useState<DatabaseStats>( {
        bans: 0,
        appeals_closed: 0,
        appeals_open: 0,
        bans_3month: 0,
        bans_6month: 0,
        bans_cidr: 0,
        bans_day: 0,
        bans_month: 0,
        bans_week: 0,
        bans_year: 0,
        filtered_words: 0,
        servers_alive: 0,
        servers_total: 0
    });

    useEffect(() => {
        const loadStats = async () => {
            try {
                const resp = await apiGetStats();
                setStats(resp)
            } catch (e) {
                console.log(`"Failed to get stats: ${e}`)
            }
        }
        loadStats()
    }, [])

    return (
        <>
            <div className={"grid-y"}>
                <div className={"cell"}>
                    <h2>Stats</h2>
                    <div className={"grid-y grid-padding-y"} >
                        <div className={"cell"}>
                            <span className={"font-bold"}>Bans Total</span> {stats.bans}
                        </div>
                        <div className={"cell"}>
                            <span className={"font-bold"}>Bans 1 Week</span> {stats.bans_week}
                        </div>
                        <div className={"cell"}>
                            <span className={"font-bold"}>Bans 1 Month</span> {stats.bans_month}
                        </div>
                        <div className={"cell"}>
                            <span className={"font-bold"}>Bans 3 Months</span> {stats.bans_3month}
                        </div><div className={"cell"}>
                        <span className={"font-bold"}>Bans 6 Months</span> {stats.bans_6month}
                    </div>
                        <div className={"cell"}>
                            <span className={"font-bold"}>Bans 1 Year</span> {stats.bans_year}
                        </div>
                        <div className={"cell"}>
                            <span className={"font-bold"}>CIDR Bans</span> {stats.bans_cidr}
                        </div>
                        <div className={"cell"}>
                            <span className={"font-bold"}>Appeals (Open)</span> {stats.appeals_open}
                        </div>
                        <div className={"cell"}>
                            <span className={"font-bold"}>Appeals (Closed)</span> {stats.appeals_closed}
                        </div>
                        <div className={"cell"}>
                            <span className={"font-bold"}>Servers (Alive)</span> {stats.servers_total} ({stats.servers_alive})
                        </div>
                    </div>
                </div>
            </div>
        </>
    )
}