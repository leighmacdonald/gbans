import React, { useEffect, useMemo, useState } from 'react';
import Grid from '@mui/material/Unstable_Grid2';
import {
    apiGetPlayerWeaponsOverall,
    apiGetProfile,
    PlayerProfile
} from '../api';
import { Nullable } from '../util/types';
import { useParams } from 'react-router';
import Stack from '@mui/material/Stack';
import { logErr } from '../util/errors';
import { createExternalLinks } from '../util/history';
import Link from '@mui/material/Link';
import Button from '@mui/material/Button';
import Chip from '@mui/material/Chip';
import { SteamIDList } from '../component/SteamIDList';
import { LoadingSpinner } from '../component/LoadingSpinner';
import { ProfileInfoBox } from '../component/ProfileInfoBox';
import SteamID from 'steamid';
import { useUserFlashCtx } from '../contexts/UserFlashCtx';
import { ContainerWithHeader } from '../component/ContainerWithHeader';
import LocalLibraryIcon from '@mui/icons-material/LocalLibrary';
import LinkIcon from '@mui/icons-material/Link';
import Box from '@mui/material/Box';
import { PageNotFound } from './PageNotFound';
import { PlayerClassStatsContainer } from '../component/PlayerClassStatsContainer';
import { PlayerStatsOverallContainer } from '../component/PlayerStatsOverallContainer';
import InsightsIcon from '@mui/icons-material/Insights';
import { WeaponsStatListContainer } from '../component/WeaponsStatListContainer';
import { useCurrentUserCtx } from '../contexts/CurrentUserCtx';
import { Login } from './Login';
import { noop } from 'lodash-es';

export const Profile = () => {
    const [profile, setProfile] = React.useState<Nullable<PlayerProfile>>(null);
    const [loading, setLoading] = React.useState<boolean>(true);
    const [error, setError] = useState('');
    const { currentUser } = useCurrentUserCtx();
    const { steam_id } = useParams();
    const { sendFlash } = useUserFlashCtx();

    useEffect(() => {
        if (!steam_id) {
            return;
        }
        const abortController = new AbortController();
        const loadProfile = async () => {
            try {
                const id = new SteamID(steam_id);
                if (!id.isValidIndividual()) {
                    setError('Invalid Steam ID');
                    return;
                }
                setLoading(true);
                apiGetProfile(id.toString(), abortController)
                    .then((response) => {
                        setProfile(response);
                    })
                    .catch(logErr)
                    .finally(() => {
                        setLoading(false);
                    });
            } catch (e) {
                setError(`Invalid Steam ID: ${steam_id}`);
                setLoading(false);
            }
        };

        loadProfile().then(noop);

        return () => abortController.abort();
    }, [sendFlash, steam_id]);

    const renderedProfile = useMemo(() => {
        if (!profile) {
            return <PageNotFound error={error} />;
        }
        return (
            <>
                <Grid xs={12} md={8}>
                    <Box width={'100%'}>
                        <ProfileInfoBox
                            profile={profile}
                            align={'flex-start'}
                        />
                    </Box>
                </Grid>
                <Grid xs={6} md={2}>
                    <ContainerWithHeader
                        title={'Status'}
                        iconLeft={<LocalLibraryIcon />}
                        marginTop={0}
                    >
                        <Stack
                            spacing={1}
                            padding={1}
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
                </Grid>
                <Grid xs={6} md={2}>
                    <SteamIDList steam_id={profile.player.steam_id} />
                </Grid>
                <Grid xs={12}>
                    {currentUser.permission_level >= 10 ? (
                        <PlayerStatsOverallContainer
                            steam_id={profile.player.steam_id}
                        />
                    ) : (
                        <Login message={'Please login to see player stats'} />
                    )}
                </Grid>
                <Grid xs={12}>
                    {currentUser.permission_level >= 10 && (
                        <PlayerClassStatsContainer
                            steam_id={profile.player.steam_id}
                        />
                    )}
                </Grid>
                <Grid xs={12}>
                    {currentUser.permission_level >= 10 && (
                        <WeaponsStatListContainer
                            title={'Overall Player Weapon Stats'}
                            icon={<InsightsIcon />}
                            fetchData={() =>
                                apiGetPlayerWeaponsOverall(
                                    profile?.player.steam_id
                                )
                            }
                        />
                    )}
                </Grid>
                <Grid xs={12}>
                    <ContainerWithHeader
                        title={'External Links'}
                        iconLeft={<LinkIcon />}
                        align={'flex-start'}
                    >
                        <Grid container spacing={1} paddingLeft={1}>
                            {createExternalLinks(profile.player.steam_id).map(
                                (l) => {
                                    return (
                                        <Grid
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
                                }
                            )}
                        </Grid>
                    </ContainerWithHeader>
                </Grid>
            </>
        );
    }, [currentUser.permission_level, error, profile]);

    return (
        <Grid container spacing={2}>
            {loading ? (
                <Grid xs={12} alignContent={'center'}>
                    <LoadingSpinner />
                </Grid>
            ) : (
                renderedProfile
            )}
        </Grid>
    );
};
