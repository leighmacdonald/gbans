import React, { useEffect } from 'react';
import Grid from '@mui/material/Grid';
import { apiGetProfile, PlayerProfile } from '../api';
import { Nullable } from '../util/types';
import { useCurrentUserCtx } from '../contexts/CurrentUserCtx';
import { useParams } from 'react-router';
import Typography from '@mui/material/Typography';
import Box from '@mui/material/Box';
import Stack from '@mui/material/Stack';

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
        <Grid container paddingTop={3}>
            {loading && (
                <Grid item xs={12}>
                    <h3>Loading Profile...</h3>
                </Grid>
            )}
            {!loading && profile && profile.player.steam_id != '' && (
                <>
                    <Grid item xs={8}>
                        <figure>
                            <img
                                src={profile.player.avatarfull}
                                alt={'Profile Avatar'}
                            />
                            <figcaption>
                                {profile.player.personaname}sdfgsadfg
                            </figcaption>
                        </figure>
                    </Grid>
                    <Grid item xs={4}>
                        <Stack spacing={3}>
                            <Box>
                                <Typography padding={3} variant={'h4'}>
                                    Friends
                                </Typography>
                            </Box>
                        </Stack>
                    </Grid>
                </>
            )}
        </Grid>
    );
};
