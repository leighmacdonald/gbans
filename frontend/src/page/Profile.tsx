import React, { useEffect } from 'react';
import Grid from '@mui/material/Grid';
import { apiGetProfile, PermissionLevel, PlayerProfile } from '../api';
import { Nullable } from '../util/types';
import { useParams } from 'react-router';
import Stack from '@mui/material/Stack';
import { logErr } from '../util/errors';
import Paper from '@mui/material/Paper';
import { FriendList } from '../component/FriendList';
import { createExternalLinks } from '../util/history';
import Link from '@mui/material/Link';
import Button from '@mui/material/Button';
import Chip from '@mui/material/Chip';
import { SteamIDList } from '../component/SteamIDList';
import { Masonry } from '@mui/lab';
import { LoadingSpinner } from '../component/LoadingSpinner';
import { MatchHistory } from '../component/MatchHistory';
import { Heading } from '../component/Heading';
import { ProfileInfoBox } from '../component/ProfileInfoBox';
import { useCurrentUserCtx } from '../contexts/CurrentUserCtx';

export const Profile = (): JSX.Element => {
    const [profile, setProfile] = React.useState<Nullable<PlayerProfile>>(null);
    const [loading, setLoading] = React.useState<boolean>(true);
    const { currentUser } = useCurrentUserCtx();
    const { steam_id } = useParams();

    useEffect(() => {
        if (!steam_id) {
            return;
        }
        setLoading(true);
        apiGetProfile(steam_id as unknown as bigint)
            .then((profile) => {
                profile && setProfile(profile);
            })
            .catch((err) => {
                logErr(err);
            })
            .finally(() => {
                setLoading(false);
            });
    }, [steam_id]);

    return (
        <Grid container paddingTop={3} spacing={3}>
            {loading && (
                <Grid item xs={12} alignContent={'center'}>
                    <LoadingSpinner />
                </Grid>
            )}
            {!loading && profile && profile.player.steam_id > 0 && (
                <>
                    <Grid item xs={8}>
                        <Stack spacing={3}>
                            <Stack direction={'row'} spacing={3}>
                                <ProfileInfoBox profile={profile} />
                                <Paper elevation={1}>
                                    <SteamIDList
                                        steam_id={profile.player.steam_id}
                                    />
                                </Paper>
                            </Stack>

                            <Paper elevation={1}>
                                <Heading>Steam Community Status</Heading>
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
                                            profile.player.economy_ban != 'none'
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
                                <Heading>External Links</Heading>
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
                            {currentUser.permission_level >=
                                PermissionLevel.Admin && (
                                <Paper elevation={1}>
                                    <Heading>Match History</Heading>
                                    <MatchHistory
                                        steam_id={profile.player.steam_id}
                                        limit={25}
                                    />
                                </Paper>
                            )}
                        </Stack>
                    </Grid>
                    <Grid item xs={4}>
                        <Stack spacing={3}>
                            <Paper elevation={1}>
                                <FriendList friends={profile?.friends || []} />
                            </Paper>
                        </Stack>
                    </Grid>
                </>
            )}
        </Grid>
    );
};
