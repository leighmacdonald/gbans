import React, { useEffect } from 'react';
import Grid from '@mui/material/Grid';
import { apiGetProfile, PlayerProfile } from '../api';
import { Nullable } from '../util/types';
import { useParams } from 'react-router';
import Typography from '@mui/material/Typography';
import Stack from '@mui/material/Stack';
import { log } from '../util/errors';
import Paper from '@mui/material/Paper';
import { FriendList } from '../component/FriendList';
import { createExternalLinks } from '../util/history';
import Link from '@mui/material/Link';
import Button from '@mui/material/Button';
import Avatar from '@mui/material/Avatar';
import Chip from '@mui/material/Chip';
import { SteamIDList } from '../component/SteamIDList';
import { Masonry } from '@mui/lab';
import { format, fromUnixTime } from 'date-fns';

export const Profile = (): JSX.Element => {
    const [profile, setProfile] = React.useState<Nullable<PlayerProfile>>(null);
    const [loading, setLoading] = React.useState<boolean>(true);

    const { steam_id } = useParams();

    useEffect(() => {
        const fetchProfile = async () => {
            log(steam_id);
            if (steam_id != '') {
                setLoading(true);
                try {
                    setProfile(await apiGetProfile(steam_id || ''));
                } catch (e) {
                    log(e);
                }
                setLoading(false);
            }
        };
        // noinspection JSIgnoredPromiseFromCall
        fetchProfile();
    }, [steam_id]);

    return (
        <Grid container paddingTop={3} spacing={3}>
            {loading && (
                <Grid item xs={12}>
                    <h3>Loading Profile...</h3>
                </Grid>
            )}
            {!loading && profile && profile.player.steam_id != '' && (
                <>
                    <Grid item xs={8}>
                        <Stack spacing={3}>
                            <Stack direction={'row'} spacing={3}>
                                <Paper elevation={1} sx={{ width: '100%' }}>
                                    <Stack
                                        direction={'row'}
                                        spacing={3}
                                        padding={3}
                                    >
                                        <Avatar
                                            variant={'square'}
                                            src={profile.player.avatarfull}
                                            alt={'Profile Avatar'}
                                            sx={{ width: 184, height: 184 }}
                                        />
                                        <Stack spacing={2}>
                                            <Typography variant={'h3'}>
                                                {profile.player.personaname}
                                            </Typography>
                                            <Typography variant={'subtitle1'}>
                                                {profile.player.realname}
                                            </Typography>
                                            <Typography variant={'body2'}>
                                                Created:{' '}
                                                {format(
                                                    fromUnixTime(
                                                        profile.player
                                                            .timecreated
                                                    ),
                                                    'yyyy-mm-dd'
                                                )}
                                            </Typography>
                                        </Stack>
                                    </Stack>
                                </Paper>
                                <Paper elevation={1}>
                                    <SteamIDList
                                        steam_id={profile.player.steam_id}
                                    />
                                </Paper>
                            </Stack>

                            <Paper elevation={1}>
                                <Stack
                                    direction="row"
                                    spacing={2}
                                    padding={2}
                                    justifyContent={'space-evenly'}
                                >
                                    <Chip
                                        color={
                                            profile.player.vac_bans > 0
                                                ? 'error'
                                                : 'success'
                                        }
                                        label={'VAC'}
                                    />
                                    <Chip
                                        color={
                                            profile.player.game_bans > 0
                                                ? 'error'
                                                : 'success'
                                        }
                                        label={'Game Ban'}
                                    />
                                    <Chip
                                        color={
                                            profile.player.economy_ban != ''
                                                ? 'error'
                                                : 'success'
                                        }
                                        label={'Economy Ban'}
                                    />
                                    <Chip
                                        color={
                                            profile.player.community_banned
                                                ? 'error'
                                                : 'success'
                                        }
                                        label={'Community Ban'}
                                    />
                                </Stack>
                            </Paper>

                            <Paper elevation={1}>
                                <Masonry columns={3} spacing={1}>
                                    {createExternalLinks(
                                        profile.player.steam_id
                                    ).map((l) => {
                                        return (
                                            <Link
                                                sx={{ width: '100%' }}
                                                component={Button}
                                                href={l.url}
                                                key={l.url}
                                                underline="none"
                                            >
                                                {l.title}
                                            </Link>
                                        );
                                    })}
                                </Masonry>
                            </Paper>
                        </Stack>
                    </Grid>
                    <Grid item xs={4}>
                        <Stack spacing={3}>
                            <Paper elevation={1}>
                                <FriendList friends={profile?.friends} />
                            </Paper>
                        </Stack>
                    </Grid>
                </>
            )}
        </Grid>
    );
};
