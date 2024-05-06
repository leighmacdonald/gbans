import BarChartIcon from '@mui/icons-material/BarChart';
import InsightsIcon from '@mui/icons-material/Insights';
import LinkIcon from '@mui/icons-material/Link';
import LocalLibraryIcon from '@mui/icons-material/LocalLibrary';
import Box from '@mui/material/Box';
import Button from '@mui/material/Button';
import Chip from '@mui/material/Chip';
import Link from '@mui/material/Link';
import Stack from '@mui/material/Stack';
import Grid from '@mui/material/Unstable_Grid2';
import { queryOptions } from '@tanstack/react-query';
import { createFileRoute, useLoaderData, useRouteContext } from '@tanstack/react-router';
import { apiGetProfile, PlayerProfile } from '../api';
import { ContainerWithHeader } from '../component/ContainerWithHeader.tsx';
import { PlayerClassStatsTable } from '../component/PlayerClassStatsTable.tsx';
import { PlayerStatsOverallContainer } from '../component/PlayerStatsOverallContainer.tsx';
import { PlayerWeaponsStatListContainer } from '../component/PlayerWeaponsStatListContainer.tsx';
import { ProfileInfoBox } from '../component/ProfileInfoBox.tsx';
import { SteamIDList } from '../component/SteamIDList.tsx';
import { createExternalLinks } from '../util/history.ts';

export const Route = createFileRoute('/_guest/profile/$steamId')({
    component: ProfilePage,
    loader: async ({ context, abortController }) => {
        const getOwnProfile = queryOptions({
            queryKey: ['ownProfile'],
            queryFn: async () => await apiGetProfile(context.auth.userSteamID, abortController)
        });

        return context.queryClient.fetchQuery(getOwnProfile);
    }
});

function ProfilePage() {
    const { userSteamID, isAuthenticated } = useRouteContext({ from: '/_guest/profile/$steamId' });
    const profile = useLoaderData({ from: '/_guest/profile/$steamId' }) as PlayerProfile;

    return (
        <Grid container spacing={2}>
            <Grid xs={12} md={8}>
                <Box width={'100%'}>
                    <ProfileInfoBox steam_id={profile.player.steam_id} />
                </Box>
            </Grid>
            <Grid xs={6} md={2}>
                <ContainerWithHeader title={'Status'} iconLeft={<LocalLibraryIcon />} marginTop={0}>
                    <Stack spacing={1} padding={1} justifyContent={'space-evenly'}>
                        <Chip color={profile.player.vac_bans > 0 ? 'error' : 'success'} label={'VAC'} />
                        <Chip color={profile.player.game_bans > 0 ? 'error' : 'success'} label={'Game Ban'} />
                        <Chip color={profile.player.economy_ban != 'none' ? 'error' : 'success'} label={'Economy Ban'} />
                        <Chip color={profile.player.community_banned ? 'error' : 'success'} label={'Community Ban'} />
                    </Stack>
                </ContainerWithHeader>
            </Grid>
            <Grid xs={6} md={2}>
                <SteamIDList steam_id={profile.player.steam_id} />
            </Grid>
            {isAuthenticated() && (userSteamID == profile.player.steam_id || !profile.settings.stats_hidden) && (
                <>
                    <Grid xs={12}>{<PlayerStatsOverallContainer steam_id={profile.player.steam_id} />}</Grid>
                    <Grid xs={12}>
                        <ContainerWithHeader title={'Player Overall Stats By Class'} iconLeft={<BarChartIcon />}>
                            <PlayerClassStatsTable steam_id={profile.player.steam_id} />
                        </ContainerWithHeader>
                    </Grid>
                    <Grid xs={12}>
                        <ContainerWithHeader title={'Overall Player Weapon Stats'} iconLeft={<InsightsIcon />}>
                            <PlayerWeaponsStatListContainer steamId={profile.player.steam_id} />
                        </ContainerWithHeader>
                    </Grid>
                </>
            )}
            <Grid xs={12}>
                <ContainerWithHeader title={'External Links'} iconLeft={<LinkIcon />}>
                    <Grid container spacing={1} paddingLeft={1}>
                        {createExternalLinks(profile.player.steam_id).map((l) => {
                            return (
                                <Grid xs={4} key={`btn-${l.url}`} padding={1}>
                                    <Button fullWidth color={'secondary'} variant={'contained'} component={Link} href={l.url} key={l.url}>
                                        {l.title}
                                    </Button>
                                </Grid>
                            );
                        })}
                    </Grid>
                </ContainerWithHeader>
            </Grid>
        </Grid>
    );
}
