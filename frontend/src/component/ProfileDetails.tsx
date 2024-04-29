import LinkIcon from '@mui/icons-material/Link';
import LocalLibraryIcon from '@mui/icons-material/LocalLibrary';
import Box from '@mui/material/Box';
import Button from '@mui/material/Button';
import Chip from '@mui/material/Chip';
import Link from '@mui/material/Link';
import Stack from '@mui/material/Stack';
import Grid from '@mui/material/Unstable_Grid2';
import { PlayerProfile } from '../api';
import { createExternalLinks } from '../util/history.ts';
import { ContainerWithHeader } from './ContainerWithHeader.tsx';
import { PlayerClassStatsContainer } from './PlayerClassStatsContainer.tsx';
import { PlayerStatsOverallContainer } from './PlayerStatsOverallContainer.tsx';
import { PlayerWeaponsStatListContainer } from './PlayerWeaponsStatListContainer.tsx';
import { ProfileInfoBox } from './ProfileInfoBox.tsx';
import { SteamIDList } from './SteamIDList.tsx';

export const ProfileDetails = ({ profile, loggedIn }: { profile: PlayerProfile; loggedIn: boolean }) => {
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
            {!profile.settings.stats_hidden && (
                <>
                    <Grid xs={12}>{loggedIn && <PlayerStatsOverallContainer steam_id={profile.player.steam_id} />}</Grid>
                    <Grid xs={12}>{loggedIn && <PlayerClassStatsContainer steam_id={profile.player.steam_id} />}</Grid>
                    <Grid xs={12}>{loggedIn && <PlayerWeaponsStatListContainer steamId={profile.player.steam_id} />}</Grid>
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
};
