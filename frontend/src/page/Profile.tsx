import React, { useEffect } from 'react';
import { apiGetProfile, PlayerProfile } from '../util/api';
import { Nullable } from '../util/types';
import { useCurrentUserCtx } from '../contexts/CurrentUserCtx';
import { RouteComponentProps } from 'react-router-dom';
import { Grid } from '@material-ui/core';

type TParams = { id: string };

export const Profile = ({
    match
}: RouteComponentProps<TParams>): JSX.Element => {
    const [profile, setProfile] = React.useState<Nullable<PlayerProfile>>(null);
    const [loading, setLoading] = React.useState<boolean>(true);
    const { currentUser } = useCurrentUserCtx();
    useEffect(() => {
        const fetchProfile = async () => {
            if (match.params.id === currentUser.player.steam_id.toString()) {
                setProfile(currentUser);
                setLoading(false);
            } else {
                setProfile(
                    (await apiGetProfile(match.params.id)) as PlayerProfile
                );
                setLoading(false);
            }
        };
        // noinspection JSIgnoredPromiseFromCall
        fetchProfile();
    }, [currentUser, match.params.id]);
    return (
        <Grid container>
            {loading && (
                <Grid item xs>
                    <h3>Loading Profile...</h3>
                </Grid>
            )}
            {!loading && profile && profile.player.steam_id > 0 && (
                <Grid item xs>
                    <figure>
                        <img
                            src={profile.player.avatarfull}
                            alt={'Profile Avatar'}
                        />
                        <figcaption>{profile.player.personaname}</figcaption>
                    </figure>
                </Grid>
            )}
        </Grid>
    );
};
