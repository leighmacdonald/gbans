import Stack from '@mui/material/Stack';
import Link from '@mui/material/Link';
import Avatar from '@mui/material/Avatar';
import Box from '@mui/material/Box';
import Typography from '@mui/material/Typography';
import React from 'react';
import SteamID from 'steamid';

export interface PersonCellProps {
    steam_id: SteamID;
    personaname: string;
    avatar: string;
}

export const PersonCell = ({
    steam_id,
    avatar,
    personaname
}: PersonCellProps) => {
    return (
        <Stack
            direction={'row'}
            alignItems={'center'}
            component={Link}
            href={`/profile/${steam_id}`}
        >
            <Avatar alt={personaname} src={avatar} variant={'square'} />
            <Box
                height={'100%'}
                alignContent={'center'}
                alignItems={'center'}
                display={'inline-block'}
                marginLeft={2}
            >
                <Typography variant={'body1'}>{personaname}</Typography>
            </Box>
        </Stack>
    );
};
