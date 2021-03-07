import React, {useEffect} from "react";
import {apiGetProfile, PlayerProfile} from "../util/api";
import {Nullable} from "../util/types";
import {useCurrentUserCtx} from "../contexts/CurrentUserCtx";

export interface ProfileProps {
    steam_id: number
}

export const Profile = ({steam_id}: ProfileProps) => {
    const [profile, setProfile] = React.useState<Nullable<PlayerProfile>>(null);
    const [loading, setLoading] = React.useState<boolean>(true);
    const {currentUser} = useCurrentUserCtx();
    useEffect(() => {
        const fetchProfile = async () => {
            if (steam_id === currentUser.player.steam_id) {
                setProfile(currentUser);
                setLoading(false)
            } else {
                try {
                    setProfile(await apiGetProfile(steam_id.toString()) as PlayerProfile)
                    setLoading(false)
                } catch (e) {
                    console.log(e)
                }
            }
        }
        // noinspection JSIgnoredPromiseFromCall
        fetchProfile();
    }, [])
    return (
        <div className="grid-container">
            <div className="grid-y grid-padding-y">
                {loading && <>
                    <h3>Loading Profile...</h3>
                </>}
                {!loading && profile && profile.player.steam_id > 0 && <>
                    <figure>
                        <img src={profile.player.avatarfull} alt={"Profile Avatar"}/>
                        <figcaption>{profile.player.personaname}</figcaption>
                    </figure>
                </>}

            </div>
        </div>
    )
}