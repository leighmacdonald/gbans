import React, {useEffect} from "react";
import {useParams} from "react-router";
import {apiGetBan, BannedPerson} from "../util/api";
import {NotNull} from "../util/types";

interface BanViewParams {
    ban_id: string
}

export const BanView = () => {
    const [loading, setLoading] = React.useState<boolean>(true);
    const [ban, setBan] = React.useState<NotNull<BannedPerson>>()
    const {ban_id} = useParams<BanViewParams>();
    useEffect(() => {
        const loadBan = async () => {
            try {
                setBan(await apiGetBan(parseInt(ban_id)) as BannedPerson)
                setLoading(false)
            } catch (e) {
                console.log(`Failed to load ban: ${e}`)
            }
        }
        loadBan()
    }, [])
    return (
        <div className="grid-container">
            {loading && !ban && <div className="grid-x grid-padding-x">
                <div className={"cell"}>
                    <h3>Loading profile...</h3>
                </div>
            </div>}
            {!loading && ban && <>
            <div className="grid-x grid-padding-x">
                <div className={"cell medium-6"}>
                    <div className={"cell"}>
                        <figure>
                            <img src={ban.person.avatarfull} alt={"Player avatar"}/>
                            <figcaption>{ban.person.personaname}</figcaption>
                        </figure>
                    </div>
                </div>
                <div className={"cell medium-6"}>

                </div>
            </div>
            <div className="grid-x grid-padding-x">
                <h3>Chat Logs</h3>
                {ban?.history_chat && ban?.history_chat.map((value, i) => {
                    return (<div className={"cell"} key={`chat-log-${i}`}>
                        <span>{value}</span>
                    </div>)
                })}
            </div>
            </>}
        </div>
    )
}