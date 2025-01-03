import { MouseEventHandler } from 'react';
import ContentCopyIcon from '@mui/icons-material/ContentCopy';
import Avatar from '@mui/material/Avatar';
import IconButton from '@mui/material/IconButton';
import Stack from '@mui/material/Stack';
import Tooltip from '@mui/material/Tooltip';
import Typography from '@mui/material/Typography';
import { useTheme } from '@mui/material/styles';
import SteamID from 'steamid';
import { useUserFlashCtx } from '../hooks/useUserFlashCtx.ts';
import { avatarHashToURL } from '../util/text.tsx';
import { ButtonLink } from './ButtonLink.tsx';

export interface PersonCellProps {
    steam_id: string;
    personaname: string;
    avatar_hash: string;
    onClick?: MouseEventHandler | undefined;
    showCopy?: boolean;
}

export const PersonCell = ({ steam_id, avatar_hash, personaname, onClick, showCopy = false }: PersonCellProps) => {
    const theme = useTheme();
    const { sendFlash } = useUserFlashCtx();

    return (
        <Stack minWidth={200} direction={'row'} alignItems={'center'}>
            {showCopy && (
                <Tooltip title={'Copy Steamid'}>
                    <IconButton
                        color={'warning'}
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
            <ButtonLink
                to={'/profile/$steamId'}
                params={{ steamId: steam_id }}
                onClick={onClick ?? undefined}
                sx={{
                    '&:hover': {
                        cursor: 'pointer',
                        backgroundColor: theme.palette.background.default
                    }
                }}
                startIcon={
                    <Avatar
                        alt={personaname}
                        src={avatarHashToURL(avatar_hash, 'small')}
                        variant={'square'}
                        sx={{ height: '32px', width: '32px' }}
                    />
                }
            >
                <Typography fontWeight={'bold'} color={theme.palette.text.primary} variant={'body1'}>
                    {personaname != '' ? personaname : steam_id}
                </Typography>
            </ButtonLink>
        </Stack>
    );
};
