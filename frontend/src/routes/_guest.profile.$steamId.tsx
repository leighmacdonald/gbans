import BarChartIcon from '@mui/icons-material/BarChart';
import InsightsIcon from '@mui/icons-material/Insights';
import LinkIcon from '@mui/icons-material/Link';
import LocalLibraryIcon from '@mui/icons-material/LocalLibrary';
import Avatar from '@mui/material/Avatar';
import Button from '@mui/material/Button';
import Chip from '@mui/material/Chip';
import Grid from '@mui/material/Grid';
import Link from '@mui/material/Link';
import Stack from '@mui/material/Stack';
import Typography from '@mui/material/Typography';
import { queryOptions } from '@tanstack/react-query';
import { createFileRoute, useLoaderData, useRouteContext } from '@tanstack/react-router';
import { format, fromUnixTime } from 'date-fns';
import { apiGetProfile } from '../api';
import { ContainerWithHeader } from '../component/ContainerWithHeader.tsx';
import { PlayerStatsOverallContainer } from '../component/PlayerStatsOverallContainer.tsx';
import { PlayerWeaponsStatListContainer } from '../component/PlayerWeaponsStatListContainer.tsx';
import { SteamIDList } from '../component/SteamIDList.tsx';
import { Title } from '../component/Title';
import { PlayerClassStatsTable } from '../component/table/PlayerClassStatsTable.tsx';
import { PlayerProfile } from '../schema/people.ts';
import { createExternalLinks } from '../util/history.ts';
import { avatarHashToURL } from '../util/text.tsx';
import { isValidSteamDate, renderDateTime } from '../util/time.ts';
import { emptyOrNullString } from '../util/types.ts';

export const Route = createFileRoute('/_guest/profile/$steamId')({
    component: ProfilePage,
    loader: async ({ context, abortController, params }) => {
        const { steamId } = params;

        return context.queryClient.fetchQuery(
            queryOptions({
                queryKey: ['profile', { steamId }],
                queryFn: async () => await apiGetProfile(steamId, abortController)
            })
        );
    }
});

function ProfilePage() {
    const { profile: userProfile, isAuthenticated } = useRouteContext({ from: '/_guest/profile/$steamId' });
    const profile = useLoaderData({ from: '/_guest/profile/$steamId' }) as PlayerProfile;

    return (
        <Grid container spacing={2}>
            {profile.player.personaname ? <Title>{profile.player.personaname}</Title> : null}
            <Grid size={{ xs: 12, md: 8 }}>
                <ContainerWithHeader title={'Profile'}>
                    <Grid container spacing={2}>
                        <Grid size={{ xs: 4 }}>
                            <Avatar
                                variant={'square'}
                                src={avatarHashToURL(profile.player.avatarhash)}
                                alt={'Profile Avatar'}
                                sx={{ width: '100%', height: '100%', minHeight: 240 }}
                            />
                        </Grid>
                        <Grid size={{ xs: 8 }}>
                            <Stack spacing={2}>
                                <Typography
                                    variant={'h3'}
                                    display="inline"
                                    style={{ wordBreak: 'break-word', whiteSpace: 'pre-line' }}
                                >
                                    {profile.player.personaname}
                                </Typography>
                                <Typography variant={'body1'}>
                                    First Seen: {renderDateTime(profile.player.created_on)}
                                </Typography>
                                {!emptyOrNullString(profile.player.locstatecode) ||
                                    (!emptyOrNullString(profile.player.loccountrycode) && (
                                        <Typography variant={'body1'}>
                                            {[profile.player.locstatecode, profile.player.loccountrycode]
                                                .filter((x) => x)
                                                .join(',')}
                                        </Typography>
                                    ))}
                                {isValidSteamDate(fromUnixTime(profile.player.timecreated)) && (
                                    <Typography variant={'body1'}>
                                        Created: {format(fromUnixTime(profile.player.timecreated), 'yyyy-MM-dd')}
                                    </Typography>
                                )}
                            </Stack>
                        </Grid>
                    </Grid>
                </ContainerWithHeader>
            </Grid>
            <Grid size={{ xs: 6, md: 2 }}>
                <ContainerWithHeader title={'Status'} iconLeft={<LocalLibraryIcon />} marginTop={0}>
                    <Stack spacing={1} padding={1} justifyContent={'space-evenly'}>
                        <Chip color={profile.player.vac_bans > 0 ? 'error' : 'success'} label={'VAC'} />
                        <Chip color={profile.player.game_bans > 0 ? 'error' : 'success'} label={'Game Ban'} />
                        <Chip
                            color={profile.player.economy_ban != 'none' ? 'error' : 'success'}
                            label={'Economy Ban'}
                        />
                        <Chip color={profile.player.community_banned ? 'error' : 'success'} label={'Community Ban'} />
                    </Stack>
                </ContainerWithHeader>
            </Grid>
            <Grid size={{ xs: 6, md: 2 }}>
                <SteamIDList steam_id={profile.player.steam_id} />
            </Grid>
            {isAuthenticated() &&
                (userProfile.steam_id == profile.player.steam_id || !profile.settings.stats_hidden) && (
                    <>
                        <Grid size={{ xs: 12 }}>
                            {<PlayerStatsOverallContainer steam_id={profile.player.steam_id} />}
                        </Grid>
                        <Grid size={{ xs: 12 }}>
                            <ContainerWithHeader title={'Player Overall Stats By Class'} iconLeft={<BarChartIcon />}>
                                <PlayerClassStatsTable steam_id={profile.player.steam_id} />
                            </ContainerWithHeader>
                        </Grid>
                        <Grid size={{ xs: 12 }}>
                            <ContainerWithHeader title={'Overall Player Weapon Stats'} iconLeft={<InsightsIcon />}>
                                <PlayerWeaponsStatListContainer steamId={profile.player.steam_id} />
                            </ContainerWithHeader>
                        </Grid>
                    </>
                )}
            <Grid size={{ xs: 128 }}>
                <ContainerWithHeader title={'External Links'} iconLeft={<LinkIcon />}>
                    <Grid container spacing={1} paddingLeft={1}>
                        {createExternalLinks(profile.player.steam_id).map((l) => {
                            return (
                                <Grid size={{ xs: 4 }} key={`btn-${l.url}`} padding={1}>
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
    );
}
