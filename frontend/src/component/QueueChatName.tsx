import Avatar from '@mui/material/Avatar';
import Typography from '@mui/material/Typography';
import { useTheme } from '@mui/material/styles';
import { avatarHashToURL } from '../util/text.tsx';
import { ButtonLink } from './ButtonLink.tsx';

export const QueueChatName = ({
    steam_id,
    personaname,
    avatarhash
}: {
    steam_id: string;
    personaname: string;
    avatarhash: string;
}) => {
    const theme = useTheme();
    return (
        <ButtonLink
            fullWidth={true}
            size={'small'}
            to={'/profile/$steamId'}
            params={{ steamId: steam_id }}
            sx={{
                justifyContent: 'flex-start',
                padding: 0,
                margin: 0,
                '&:hover': {
                    cursor: 'pointer',
                    backgroundColor: theme.palette.background.default
                }
            }}
            startIcon={
                <Avatar
                    alt={personaname}
                    src={avatarHashToURL(avatarhash, 'small')}
                    variant={'square'}
                    sx={{ height: '16px', width: '16px' }}
                />
            }
        >
            <Typography fontWeight={'bold'} color={theme.palette.text.primary} variant={'body1'}>
                {personaname != '' ? personaname : steam_id}
            </Typography>
        </ButtonLink>
    );
};
