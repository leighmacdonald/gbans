import React, { useEffect, useState } from 'react';
import PregnantWomanIcon from '@mui/icons-material/PregnantWoman';
import Avatar from '@mui/material/Avatar';
import Stack from '@mui/material/Stack';
import Typography from '@mui/material/Typography';
import { format, fromUnixTime } from 'date-fns';
import { apiGetProfile, PlayerProfile } from '../api';
import { logErr } from '../util/errors';
import { avatarHashToURL, isValidSteamDate } from '../util/text';
import { ContainerWithHeader } from './ContainerWithHeader';
import { LoadingPlaceholder } from './LoadingPlaceholder';

export const ProfileInfoBox = ({ steam_id }: { steam_id: string }) => {
    const [profile, setProfile] = useState<PlayerProfile>();

    useEffect(() => {
        apiGetProfile(steam_id)
            .then((profile) => {
                setProfile(profile);
            })
            .catch((reason) => {
                logErr(reason);
            });
    }, [steam_id]);

    return (
        <ContainerWithHeader
            title={'Profile'}
            iconLeft={<PregnantWomanIcon />}
            marginTop={0}
        >
            {profile == undefined ? (
                <LoadingPlaceholder />
            ) : (
                <Stack direction={'row'} spacing={3} marginTop={0}>
                    <Avatar
                        variant={'square'}
                        src={avatarHashToURL(profile.player.avatarhash)}
                        alt={'Profile Avatar'}
                        sx={{ width: 160, height: 160 }}
                    />
                    <Stack spacing={2} paddingTop={0}>
                        <Typography variant={'h1'}>
                            {profile.player.personaname}
                        </Typography>
                        <Typography variant={'subtitle1'}>
                            {profile.player.realname}
                        </Typography>
                        <Typography variant={'body1'}>
                            {[
                                profile.player.locstatecode,
                                profile.player.loccountrycode
                            ]
                                .filter((x) => x)
                                .join(',')}
                        </Typography>
                        {isValidSteamDate(
                            fromUnixTime(profile.player.timecreated)
                        ) && (
                            <Typography variant={'body1'}>
                                Created:{' '}
                                {format(
                                    fromUnixTime(profile.player.timecreated),
                                    'yyyy-MM-dd'
                                )}
                            </Typography>
                        )}
                    </Stack>
                </Stack>
            )}
        </ContainerWithHeader>
    );
};
