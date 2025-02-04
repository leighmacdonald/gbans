import { FormEvent, useCallback, useState } from 'react';
import ScrollableFeed from 'react-scrollable-feed';
import HourglassBottomIcon from '@mui/icons-material/HourglassBottom';
import ManageAccountsIcon from '@mui/icons-material/ManageAccounts';
import PersonIcon from '@mui/icons-material/Person';
import PersonOffIcon from '@mui/icons-material/PersonOff';
import SendIcon from '@mui/icons-material/Send';
import Avatar from '@mui/material/Avatar';
import Collapse from '@mui/material/Collapse';
import IconButton from '@mui/material/IconButton';
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
    const { messages, isReady, sendMessage, showChat, users } = useQueueCtx();
    const { hasPermission, profile } = useAuth();
    const [showPeople, setShowPeople] = useState<boolean>(false);
    const [msg, setMsg] = useState<string>('');
    const [sending, setSending] = useState(false);

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
    const isMod = hasPermission(PermissionLevel.Moderator);

    if (!isMod || profile.playerqueue_chat_status == 'noaccess') {
        return <></>;
    }

    return (
        <Collapse in={showChat}>
            <Grid container spacing={1} sx={{ marginBottom: 3 }}>
                <Grid xs={showPeople ? 10 : 12}>
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
                                <ScrollableFeed>
                                    {messages.map((message, i) => {
                                        return (
                                            <ChatRow
                                                message={message}
                                                key={`${message.message_id}-${i}`}
                                                showControls={isMod}
                                            />
                                        );
                                    })}
                                </ScrollableFeed>
                            )}
                        </Stack>

                        <form onSubmit={onSubmit}>
                            <Stack direction={'row'} spacing={1} padding={2}>
                                <TextField
                                    disabled={sending || !isReady || profile.playerqueue_chat_status != 'readwrite'}
                                    onSubmit={onSubmit}
                                    fullWidth={true}
                                    size={'small'}
                                    name={'msg'}
                                    value={msg}
                                    onChange={(event) => {
                                        setMsg(event.target.value);
                                    }}
                                />

                                <IconButton
                                    color={'primary'}
                                    onClick={() => {
                                        setShowPeople((prevState) => !prevState);
                                    }}
                                >
                                    {showPeople ? <PersonIcon /> : <PersonOffIcon />}
                                </IconButton>
                                <SubmitButton
                                    label={'Send'}
                                    startIcon={sending ? <HourglassBottomIcon /> : <SendIcon />}
                                    size={'small'}
                                />
                            </Stack>
                        </form>
                    </Paper>
                </Grid>
                {showPeople && (
                    <Grid xs={2}>
                        <Paper>
                            <Stack
                                maxHeight={237}
                                minHeight={237}
                                overflow={'auto'}
                                sx={{ overflowX: 'hidden' }}
                                direction={'column'}
                                padding={1}
                            >
                                {users.map((u) => {
                                    return (
                                        <ChatName
                                            key={`memberlist-${u.steam_id}`}
                                            personaname={u.name}
                                            steam_id={u.steam_id}
                                            avatarhash={u.hash}
                                        />
                                    );
                                })}
                            </Stack>
                        </Paper>
                    </Grid>
                )}
            </Grid>
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

const ChatRow = ({ message, showControls }: { message: ServerQueueMessage; showControls: boolean }) => {
    return (
        <Grid container key={`${message.message_id}-id`} spacing={1} paddingLeft={1} paddingRight={1}>
            <Grid xs={2} alignItems="flex-start" justifyContent="flex-start">
                <ChatName
                    personaname={message.personaname}
                    steam_id={message.steam_id}
                    avatarhash={message.avatarhash}
                />
            </Grid>
            <Grid xs={10}>
                <Stack direction={'row'} spacing={1}>
                    {showControls && (
                        <IconButton
                            color={'primary'}
                            sx={{
                                size: '10',
                                padding: 0,
                                borderLeft: '1px solid #666',
                                borderRadius: 0,
                                paddingLeft: 1
                            }}
                        >
                            <ManageAccountsIcon color={'error'} />
                        </IconButton>
                    )}
                    <Typography variant="body1" color="text" sx={{ borderLeft: '1px solid #666', paddingLeft: 1 }}>
                        {message.body_md}
                    </Typography>
                </Stack>
            </Grid>
        </Grid>
    );
};
