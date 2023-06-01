import React, { useEffect, useMemo } from 'react';
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
import { LoadingSpinner } from '../component/LoadingSpinner';
import { MatchHistory } from '../component/MatchHistory';
import RestoreIcon from '@mui/icons-material/Restore';
import { ProfileInfoBox } from '../component/ProfileInfoBox';
import { useCurrentUserCtx } from '../contexts/CurrentUserCtx';
import SteamID from 'steamid';
import { useUserFlashCtx } from '../contexts/UserFlashCtx';
import { ContainerWithHeader } from '../component/ContainerWithHeader';
import LocalLibraryIcon from '@mui/icons-material/LocalLibrary';
import LinkIcon from '@mui/icons-material/Link';
import Box from '@mui/material/Box';

export const Profile = () => {
    const [profile, setProfile] = React.useState<Nullable<PlayerProfile>>(null);
    const [loading, setLoading] = React.useState<boolean>(true);
    const { currentUser } = useCurrentUserCtx();
    const { steam_id } = useParams();
    const { sendFlash } = useUserFlashCtx();

    const matchOpts = useMemo(() => {
        return { steam_id: profile?.player.steam_id, limit: 25 };
    }, [profile?.player.steam_id]);

    useEffect(() => {
        if (!steam_id) {
            return;
        }
        const id = new SteamID(steam_id);
        if (!id.isValidIndividual()) {
            return;
        }
        setLoading(true);
        apiGetProfile(id.toString())
            .then((response) => {
                if (!response.status || !response.result) {
                    sendFlash('error', 'Failed to load profile');
                    return;
                }
                response && setProfile(response.result);
            })
            .catch(logErr)
            .finally(() => {
                setLoading(false);
            });
    }, [sendFlash, steam_id]);

    const renderedProfile = useMemo(() => {
        if (!profile) {
            return <></>;
        }
        return (
            <>
                <Grid item xs={8}>
                    <Stack spacing={3}>
                        <Stack direction={'row'} spacing={3} marginTop={0}>
                            <Box width={'100%'}>
                                <ProfileInfoBox
                                    profile={profile}
                                    align={'flex-start'}
                                />
                            </Box>
                            <SteamIDList steam_id={profile.player.steam_id} />
                        </Stack>
                        <ContainerWithHeader
                            title={'Steam Community Status'}
                            iconLeft={<LocalLibraryIcon />}
                            marginTop={0}
                        >
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
                        </ContainerWithHeader>

                        <ContainerWithHeader
                            title={'External Links'}
                            iconLeft={<LinkIcon />}
                            align={'flex-start'}
                        >
                            <Grid container spacing={1} paddingLeft={1}>
                                {createExternalLinks(
                                    profile.player.steam_id
                                ).map((l) => {
                                    return (
                                        <Grid
                                            item
                                            xs={4}
                                            key={`btn-${l.url}`}
                                            padding={1}
                                        >
                                            <Button
                                                fullWidth
                                                color={'secondary'}
                                                variant={'contained'}
                                                component={Link}
                                                href={l.url}
                                                key={l.url}
                                            >
                                                {l.title}
                                            </Button>
                                        </Grid>
                                    );
                                })}
                            </Grid>
                        </ContainerWithHeader>

                        {currentUser.permission_level >=
                            PermissionLevel.Admin && (
                            <ContainerWithHeader
                                title={'Match History'}
                                iconLeft={<RestoreIcon />}
                                marginTop={0}
                            >
                                <MatchHistory opts={matchOpts} />
                            </ContainerWithHeader>
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
        );
    }, [currentUser.permission_level, matchOpts, profile]);

    return (
        <Grid container paddingTop={3} spacing={3}>
            {loading ? (
                <Grid item xs={12} alignContent={'center'}>
                    <LoadingSpinner />
                </Grid>
            ) : (
                renderedProfile
            )}
        </Grid>
    );
};
