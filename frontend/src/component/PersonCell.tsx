import { MouseEventHandler } from 'react';
import Avatar from '@mui/material/Avatar';
import Box from '@mui/material/Box';
import Stack from '@mui/material/Stack';
import Tooltip from '@mui/material/Tooltip';
import Typography from '@mui/material/Typography';
import { useTheme } from '@mui/material/styles';
import { useNavigate } from '@tanstack/react-router';
import { avatarHashToURL } from '../util/text.tsx';

export interface PersonCellProps {
    steam_id: string;
    personaname: string;
    avatar_hash: string;
    onClick?: MouseEventHandler | undefined;
}

export const PersonCell = ({
    steam_id,
    avatar_hash,
    personaname,
    onClick
}: PersonCellProps) => {
    const navigate = useNavigate();
    const theme = useTheme();

    return (
        <Stack
            minWidth={200}
            direction={'row'}
            alignItems={'center'}
            onClick={
                onClick != undefined
                    ? onClick
                    : () => {
                          navigate(`/profile/${steam_id}`);
                      }
            }
            sx={{
                '&:hover': {
                    cursor: 'pointer',
                    backgroundColor: theme.palette.background.default
                }
            }}
        >
            <Tooltip title={personaname}>
                <>
                    <Avatar
                        alt={personaname}
                        src={avatarHashToURL(avatar_hash, 'small')}
                        variant={'square'}
                        sx={{ height: '32px', width: '32px' }}
                    />
                </>
            </Tooltip>

            <Box
                height={'100%'}
                alignContent={'center'}
                alignItems={'center'}
                display={'inline-block'}
                marginLeft={personaname == '' ? 0 : 2}
            >
                <Typography variant={'body1'}>{personaname}</Typography>
            </Box>
        </Stack>
    );
};
