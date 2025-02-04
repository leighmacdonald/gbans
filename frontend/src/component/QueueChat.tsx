import { FormEvent, useCallback, useState } from 'react';
import HourglassBottomIcon from '@mui/icons-material/HourglassBottom';
import PersonIcon from '@mui/icons-material/Person';
import PersonOffIcon from '@mui/icons-material/PersonOff';
import SendIcon from '@mui/icons-material/Send';
import Collapse from '@mui/material/Collapse';
import IconButton from '@mui/material/IconButton';
import Paper from '@mui/material/Paper';
import Stack from '@mui/material/Stack';
import TextField from '@mui/material/TextField';
import Grid from '@mui/material/Unstable_Grid2';
import { PermissionLevel } from '../api';
import { useAuth } from '../hooks/useAuth.ts';
import { useQueueCtx } from '../hooks/useQueueCtx.ts';
import { QueueChatMessageContainer } from './QueueChatMessageContainer.tsx';
import { QueueChatName } from './QueueChatName.tsx';
import { SubmitButton } from './modal/Buttons.tsx';

export const QueueChat = () => {
    const { isReady, sendMessage, showChat, users } = useQueueCtx();
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
                        <Stack maxHeight={200} minHeight={200} overflow={'auto'} sx={{ overflowX: 'hidden' }}>
                            <QueueChatMessageContainer showControls={isMod} />
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
                                        <QueueChatName
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
