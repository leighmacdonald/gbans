import { MouseEventHandler } from 'react';
import ContentCopyIcon from '@mui/icons-material/ContentCopy';
import Avatar from '@mui/material/Avatar';
import Box from '@mui/material/Box';
import IconButton from '@mui/material/IconButton';
import Stack from '@mui/material/Stack';
import Tooltip from '@mui/material/Tooltip';
import Typography from '@mui/material/Typography';
import { useTheme } from '@mui/material/styles';
import { useNavigate } from '@tanstack/react-router';
import SteamID from 'steamid';
import { useUserFlashCtx } from '../hooks/useUserFlashCtx.ts';
import { avatarHashToURL } from '../util/text.tsx';

export interface PersonCellProps {
    steam_id: string;
    personaname: string;
    avatar_hash: string;
    onClick?: MouseEventHandler | undefined;
    showCopy?: boolean;
}

export const PersonCell = ({ steam_id, avatar_hash, personaname, onClick, showCopy = false }: PersonCellProps) => {
    const navigate = useNavigate();
    const theme = useTheme();
    const { sendFlash } = useUserFlashCtx();

    return (
        <Stack
            minWidth={200}
            direction={'row'}
            alignItems={'center'}
            onClick={
                onClick != undefined
                    ? onClick
                    : async () => {
                          await navigate({ to: `/profile/${steam_id}` });
                      }
            }
            sx={{
                '&:hover': {
                    cursor: 'pointer',
                    backgroundColor: theme.palette.background.default
                }
            }}
        >
            {showCopy && (
                <Tooltip title={'Copy Steamid'}>
                    <IconButton
                        onClick={async (event) => {
                            event.preventDefault();
                            event.stopPropagation();
                            const sid = new SteamID(steam_id);
                            await navigator.clipboard.writeText(sid.toString());
                            sendFlash('success', `Copied to clipboard: ${sid.toString()}`);
                        }}
                    >
                        <ContentCopyIcon />
                    </IconButton>
                </Tooltip>
            )}
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
