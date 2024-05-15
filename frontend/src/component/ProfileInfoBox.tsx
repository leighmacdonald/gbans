import PregnantWomanIcon from '@mui/icons-material/PregnantWoman';
import Avatar from '@mui/material/Avatar';
import Box from '@mui/material/Box';
import Typography from '@mui/material/Typography';
import Grid from '@mui/material/Unstable_Grid2';
import { useQuery } from '@tanstack/react-query';
import { format, fromUnixTime } from 'date-fns';
import { apiGetProfile } from '../api';
import { ErrorCode } from '../error.tsx';
import { avatarHashToURL, isValidSteamDate, renderDateTime } from '../util/text.tsx';
import { emptyOrNullString } from '../util/types.ts';
import { ContainerWithHeader } from './ContainerWithHeader';
import { ErrorDetails } from './ErrorDetails.tsx';
import { LoadingPlaceholder } from './LoadingPlaceholder';

export const ProfileInfoBox = ({ steam_id }: { steam_id: string }) => {
    const { data: profile, isLoading } = useQuery({
        queryKey: ['profile', { steam_id }],
        queryFn: async () => await apiGetProfile(steam_id)
    });

    if (isLoading) {
        return <LoadingPlaceholder />;
    }

    if (!profile) {
        return <ErrorDetails error={ErrorCode.Unknown} />;
    }

    return (
        <ContainerWithHeader title={'Profile'} iconLeft={<PregnantWomanIcon />} marginTop={0}>
            <Grid container spacing={1}>
                <Grid xs={12}>
                    <Avatar
                        variant={'square'}
                        src={avatarHashToURL(profile.player.avatarhash)}
                        alt={'Profile Avatar'}
                        sx={{ width: '100%', height: '100%' }}
                    />
                </Grid>
                <Grid xs={12}>
                    <Box>
                        <Typography
                            variant={'h3'}
                            display="inline"
                            style={{ wordBreak: 'break-word', whiteSpace: 'pre-line' }}
                        >
                            {profile.player.personaname + profile.player.personaname}
                        </Typography>
                    </Box>
                </Grid>

                <Grid xs={12}>
                    <Typography variant={'body1'}>First Seen: {renderDateTime(profile.player.created_on)}</Typography>
                </Grid>

                {!emptyOrNullString(profile.player.locstatecode) ||
                    (!emptyOrNullString(profile.player.loccountrycode) && (
                        <Grid xs={12}>
                            <Typography variant={'body1'}>
                                {[profile.player.locstatecode, profile.player.loccountrycode]
                                    .filter((x) => x)
                                    .join(',')}
                            </Typography>
                        </Grid>
                    ))}

                {isValidSteamDate(fromUnixTime(profile.player.timecreated)) && (
                    <Grid xs={12}>
                        <Typography variant={'body1'}>
                            Created: {format(fromUnixTime(profile.player.timecreated), 'yyyy-MM-dd')}
                        </Typography>
                    </Grid>
                )}
            </Grid>
        </ContainerWithHeader>
    );
};
