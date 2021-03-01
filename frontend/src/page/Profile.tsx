import React, {useEffect} from "react";
import {PlayerSummary} from "../component/PlayerBanForm";
import {apiCall} from "../util/network";

export interface ProfileProps {
    steam_id: number
}

export const Profile = ({steam_id}: ProfileProps) => {
    const [profile, setProfile] = React.useState<PlayerSummary>({personaname: "", steam_id: 0});
    useEffect(() => {
        const fetchProfile = async () => {
            const resp = await apiCall<PlayerSummary>(`/api/profiles/${steam_id}`, "GET");
            if (!resp.status) {
                // TODO Add flash / redirect to login
                console.log("Bad fetch profile response")
                return
            }
            setProfile(resp.json as PlayerSummary)
            console.log(resp.json)
        }
        // noinspection JSIgnoredPromiseFromCall
        fetchProfile();
    }, [])
    return (
        <div className="grid-container">
            <div className="grid-y grid-padding-y">
                <figure>
                    <img src={profile.avatarfull} alt={"Profile Avatar"}/>
                    <figcaption>{profile.personaname}</figcaption>
                </figure>

            </div>
        </div>
    )
}