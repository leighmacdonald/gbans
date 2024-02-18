import { useParams } from 'react-router';
import LinkIcon from '@mui/icons-material/Link';
import LocalLibraryIcon from '@mui/icons-material/LocalLibrary';
import Box from '@mui/material/Box';
import Button from '@mui/material/Button';
import Chip from '@mui/material/Chip';
import Link from '@mui/material/Link';
import Stack from '@mui/material/Stack';
import Typography from '@mui/material/Typography';
import Grid from '@mui/material/Unstable_Grid2';
import { ContainerWithHeader } from '../component/ContainerWithHeader';
import { LoadingPlaceholder } from '../component/LoadingPlaceholder';
import { PlayerClassStatsContainer } from '../component/PlayerClassStatsContainer';
import { PlayerStatsOverallContainer } from '../component/PlayerStatsOverallContainer';
import { PlayerWeaponsStatListContainer } from '../component/PlayerWeaponsStatListContainer';
import { ProfileInfoBox } from '../component/ProfileInfoBox';
import { SteamIDList } from '../component/SteamIDList';
import { useCurrentUserCtx } from '../hooks/useCurrentUserCtx.ts';
import { useProfile } from '../hooks/useProfile';
import { createExternalLinks } from '../util/history';
import { LoginPage } from './LoginPage';

export const ProfilePage = () => {
    const { currentUser } = useCurrentUserCtx();
    const { steam_id } = useParams();
    const { data, loading, error } = useProfile(steam_id ?? '');

    if (loading) {
        return (
            <Grid container spacing={2}>
                <Grid xs={12} alignContent={'center'}>
                    <LoadingPlaceholder />
                </Grid>
            </Grid>
        );
    }

    if (!data?.player && error) {
        return (
            <Grid container spacing={2}>
                <Grid xs={12} alignContent={'center'}>
                    <Typography align={'center'}>{error.message}</Typography>
                </Grid>
            </Grid>
        );
    }

    return data?.player ? (
        <Grid container spacing={2}>
            <Grid xs={12} md={8}>
                <Box width={'100%'}>
                    <ProfileInfoBox steam_id={data.player.steam_id} />
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
                                data.player.vac_bans > 0 ? 'error' : 'success'
                            }
                            label={'VAC'}
                        />
                        <Chip
                            color={
                                data.player.game_bans > 0 ? 'error' : 'success'
                            }
                            label={'Game Ban'}
                        />
                        <Chip
                            color={
                                data.player.economy_ban != 'none'
                                    ? 'error'
                                    : 'success'
                            }
                            label={'Economy Ban'}
                        />
                        <Chip
                            color={
                                data.player.community_banned
                                    ? 'error'
                                    : 'success'
                            }
                            label={'Community Ban'}
                        />
                    </Stack>
                </ContainerWithHeader>
            </Grid>
            <Grid xs={6} md={2}>
                <SteamIDList steam_id={data.player.steam_id} />
            </Grid>
            {!data.settings.stats_hidden && (
                <>
                    <Grid xs={12}>
                        {currentUser.permission_level >= 10 ? (
                            <PlayerStatsOverallContainer
                                steam_id={data.player.steam_id}
                            />
                        ) : (
                            <LoginPage
                                message={'Please login to see player stats'}
                            />
                        )}
                    </Grid>
                    <Grid xs={12}>
                        {currentUser.permission_level >= 10 && (
                            <PlayerClassStatsContainer
                                steam_id={data.player.steam_id}
                            />
                        )}
                    </Grid>
                    <Grid xs={12}>
                        {currentUser.permission_level >= 10 && (
                            <PlayerWeaponsStatListContainer
                                steamId={data.player.steam_id}
                            />
                        )}
                    </Grid>
                </>
            )}
            <Grid xs={12}>
                <ContainerWithHeader
                    title={'External Links'}
                    iconLeft={<LinkIcon />}
                >
                    <Grid container spacing={1} paddingLeft={1}>
                        {createExternalLinks(data.player.steam_id).map((l) => {
                            return (
                                <Grid xs={4} key={`btn-${l.url}`} padding={1}>
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
            </Grid>
        </Grid>
    ) : (
        <></>
    );
};

export default ProfilePage;
