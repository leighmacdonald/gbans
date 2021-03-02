import React, {useEffect} from "react";
import {apiGetProfile, Person} from "../util/api";
import {Nullable} from "../util/types";

export interface ProfileProps {
    steam_id: number
}

export const Profile = ({steam_id}: ProfileProps) => {
    const [profile, setProfile] = React.useState<Nullable<Person>>(null);
    const [loading, setLoading] = React.useState<Nullable<boolean>>(true);
    useEffect(() => {
        const fetchProfile = async () => {
            try {
                setProfile(await apiGetProfile(steam_id.toString()) as Person)
                setLoading(false)
            } catch (e) {
                console.log(e)
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
                {!loading && profile && <>
                    <figure>
                        <img src={profile.avatarfull} alt={"Profile Avatar"}/>
                        <figcaption>{profile.personaname}</figcaption>
                    </figure>
                </>}

            </div>
        </div>
    )
}