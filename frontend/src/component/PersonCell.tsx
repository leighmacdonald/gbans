import Stack from '@mui/material/Stack';
import Avatar from '@mui/material/Avatar';
import Box from '@mui/material/Box';
import Typography from '@mui/material/Typography';
import React from 'react';
import { useNavigate } from 'react-router-dom';

export interface PersonCellProps {
    steam_id: string;
    personaname: string;
    avatar: string;
}

export const PersonCell = ({
    steam_id,
    avatar,
    personaname
}: PersonCellProps) => {
    const navigate = useNavigate();
    return (
        <Stack direction={'row'} alignItems={'center'}>
            <Avatar
                alt={personaname}
                src={avatar}
                variant={'square'}
                sx={{ height: '32px', width: '32px' }}
            />
            <Box
                height={'100%'}
                alignContent={'center'}
                alignItems={'center'}
                display={'inline-block'}
                marginLeft={2}
                onClick={() => {
                    navigate(`/profile/${steam_id}`);
                }}
            >
                <Typography variant={'body1'}>{personaname}</Typography>
            </Box>
        </Stack>
    );
};
