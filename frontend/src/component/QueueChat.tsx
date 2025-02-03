import { FormEvent, useCallback, useEffect, useRef, useState } from 'react';
import HourglassBottomIcon from '@mui/icons-material/HourglassBottom';
import SendIcon from '@mui/icons-material/Send';
import Avatar from '@mui/material/Avatar';
import Collapse from '@mui/material/Collapse';
import Paper from '@mui/material/Paper';
import Stack from '@mui/material/Stack';
import TextField from '@mui/material/TextField';
import Typography from '@mui/material/Typography';
import Grid from '@mui/material/Unstable_Grid2';
import { useTheme } from '@mui/material/styles';
import { PermissionLevel, ServerQueueMessage } from '../api';
import { useAuth } from '../hooks/useAuth.ts';
import { useQueueCtx } from '../hooks/useQueueCtx.ts';
import { avatarHashToURL } from '../util/text.tsx';
import { ButtonLink } from './ButtonLink.tsx';
import { LoadingPlaceholder } from './LoadingPlaceholder.tsx';
import { SubmitButton } from './modal/Buttons.tsx';

export const QueueChat = () => {
    const { messages, isReady, sendMessage, showChat } = useQueueCtx();
    const { hasPermission } = useAuth();
    const [msg, setMsg] = useState<string>('');
    const [sending, setSending] = useState(false);
    const messagesEndRef = useRef<null | HTMLDivElement>(null);

    const scrollToBottom = () => {
        if (messagesEndRef.current) {
            messagesEndRef.current.scrollIntoView({ behavior: 'smooth' });
        }
    };

    useEffect(() => {
        scrollToBottom();
    }, [messages]);

    const onSubmit = useCallback(
        async (event: FormEvent<HTMLFormElement | HTMLDivElement>) => {
            event.preventDefault();

            if (msg.length > 0) {
                setSending(true);
                sendMessage(msg);
                setMsg('');
                setSending(false);
            }
        },
        [msg]
    );

    if (!hasPermission(PermissionLevel.Moderator)) {
        return <></>;
    }

    return (
        <Collapse in={showChat}>
            <Paper>
                <Stack
                    maxHeight={200}
                    minHeight={200}
                    overflow={'auto'}
                    sx={{ overflowX: 'hidden' }}
                    direction={'column'}
                    padding={1}
                >
                    {!isReady ? (
                        <LoadingPlaceholder></LoadingPlaceholder>
                    ) : (
                        messages.map((message, i) => {
                            return <ChatRow message={message} key={`${message.id}-${i}`} />;
                        })
                    )}
                    <span ref={messagesEndRef} key={'hi'} />
                </Stack>

                <Grid container padding={1}>
                    <Grid xs={12}>
                        <form onSubmit={onSubmit}>
                            <Stack direction={'row'} spacing={1}>
                                <TextField
                                    disabled={sending || !isReady}
                                    onSubmit={onSubmit}
                                    fullWidth={true}
                                    size={'small'}
                                    name={'msg'}
                                    value={msg}
                                    onChange={(event) => {
                                        setMsg(event.target.value);
                                    }}
                                />
                                <SubmitButton
                                    label={'Send'}
                                    startIcon={sending ? <HourglassBottomIcon /> : <SendIcon />}
                                    size={'small'}
                                />
                            </Stack>
                        </form>
                    </Grid>
                </Grid>
            </Paper>
        </Collapse>
    );
};

const ChatName = ({
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
const ChatRow = ({ message }: { message: ServerQueueMessage }) => {
    return (
        <Grid container key={`${message.id}-id`} spacing={1}>
            <Grid xs={2} alignItems="flex-start" justifyContent="flex-start">
                <ChatName
                    personaname={message.personaname}
                    steam_id={message.steam_id}
                    avatarhash={message.avatarhash}
                />
            </Grid>
            <Grid xs={10}>
                <Typography variant="body1" color="text">
                    {message.body_md}
                </Typography>
            </Grid>
        </Grid>
    );
};
