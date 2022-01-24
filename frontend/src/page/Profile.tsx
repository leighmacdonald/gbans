import React, { useEffect } from 'react';
import Grid from '@mui/material/Grid';
import { apiGetProfile, PlayerProfile } from '../api';
import { Nullable } from '../util/types';
import { useCurrentUserCtx } from '../contexts/CurrentUserCtx';
import { useParams } from 'react-router';

export const Profile = (): JSX.Element => {
    const [profile, setProfile] = React.useState<Nullable<PlayerProfile>>(null);
    const [loading, setLoading] = React.useState<boolean>(true);
    const { currentUser } = useCurrentUserCtx();
    const { id } = useParams();
    useEffect(() => {
        const fetchProfile = async () => {
            if (id === currentUser.player.steam_id.toString()) {
                setProfile(currentUser);
                setLoading(false);
            } else {
                setProfile((await apiGetProfile(id || '')) as PlayerProfile);
                setLoading(false);
            }
        };
        // noinspection JSIgnoredPromiseFromCall
        fetchProfile();
    }, [currentUser, id]);
    return (
        <Grid container>
            {loading && (
                <Grid item xs>
                    <h3>Loading Profile...</h3>
                </Grid>
            )}
            {!loading && profile && profile.player.steam_id != '' && (
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
