import { FormEvent, useCallback, useEffect, useRef, useState } from 'react';
import useWebSocket, { ReadyState } from 'react-use-websocket';
import HourglassBottomIcon from '@mui/icons-material/HourglassBottom';
import SendIcon from '@mui/icons-material/Send';
import Stack from '@mui/material/Stack';
import TextField from '@mui/material/TextField';
import Typography from '@mui/material/Typography';
import Grid from '@mui/material/Unstable_Grid2';
import { defaultAvatarHash, Operation, PermissionLevel, queuePayload, ServerQueueMessage, websocketURL } from '../api';
import { useAuth } from '../hooks/useAuth.ts';
import { transformCreatedOnDate } from '../util/time.ts';
import { ContainerWithHeader } from './ContainerWithHeader.tsx';
import { LoadingPlaceholder } from './LoadingPlaceholder.tsx';
import { SubmitButton } from './modal/Buttons.tsx';

export const QueueChat = () => {
    const [isReady, setIsReady] = useState(false);
    const [messages, setMessages] = useState<ServerQueueMessage[]>([]);
    const [msg, setMsg] = useState<string>('');
    const [sending, setSending] = useState(false);
    const messagesEndRef = useRef<null | HTMLDivElement>(null);
    const { hasPermission } = useAuth();

    const scrollToBottom = () => {
        messagesEndRef.current?.scrollIntoView({ behavior: 'smooth' });
    };

    useEffect(() => {
        scrollToBottom();
    }, [messages]);

    const { readyState, sendJsonMessage } = useWebSocket(websocketURL(), {
        filter: (message) => {
            const req = JSON.parse(message.data) as queuePayload<ServerQueueMessage>;
            if (req.op == Operation.MessageRecv) {
                setMessages((prev) => [...prev, transformCreatedOnDate(req.payload)]);

                return true;
            }
            return false;
        },
        share: true
    });

    useEffect(() => {
        switch (readyState) {
            case ReadyState.OPEN:
                setIsReady(true);
                setMessages((prevState) => [
                    ...prevState,
                    {
                        created_on: new Date(),
                        body_md: 'Connected to queue',
                        avatarhash: '',
                        permission_level: PermissionLevel.Reserved,
                        personaname: 'SYSTEM',
                        steam_id: 'SYSTEM'
                    } as ServerQueueMessage
                ]);
                break;
            case ReadyState.CLOSED:
                setMessages((prevState) => [
                    ...prevState,
                    {
                        created_on: new Date(),
                        body_md: 'Disconnected from queue',
                        avatarhash: '',
                        permission_level: PermissionLevel.Reserved,
                        personaname: 'SYSTEM',
                        steam_id: 'SYSTEM'
                    } as ServerQueueMessage
                ]);
                break;
            default:
                setIsReady(false);
        }
    }, [readyState]);

    const onSubmit = useCallback(
        async (event: FormEvent<HTMLFormElement | HTMLDivElement>) => {
            event.preventDefault();

            if (msg.length > 0) {
                setSending(true);
                sendJsonMessage<queuePayload<ServerQueueMessage>>({
                    op: Operation.MessageSend,
                    payload: {
                        body_md: msg,
                        personaname: 'queue',
                        permission_level: PermissionLevel.Reserved,
                        avatarhash: defaultAvatarHash,
                        steam_id: 'queue',
                        created_on: new Date(),
                        id: '0194ba54-f9cc-7d10-bbe6-1296e6aa939a'
                    }
                });
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
        <ContainerWithHeader title={'Queue Chat'}>
            <Grid
                container
                maxHeight={200}
                minHeight={200}
                overflow={'hidden'}

                //direction="column"
                // sx={{
                //     justifyContent: 'flex-start',
                //     alignItems: 'flex-start'
                // }}
            >
                {!isReady && <LoadingPlaceholder></LoadingPlaceholder>}
                {isReady &&
                    messages.map((message) => {
                        return (
                            <Grid
                                xs={12}
                                key={`msg-id-${message.avatarhash}-${message.created_on}-${message.body_md}`}
                                overflow={'scroll'}
                            >
                                <Grid container>
                                    <Grid xs={2}>
                                        <Typography variant="body2" color="textSecondary">
                                            {message.steam_id}:
                                        </Typography>
                                    </Grid>
                                    <Grid xs={10}>
                                        <Typography variant="body1" color="text">
                                            {message.id} || {message.body_md}
                                        </Typography>
                                    </Grid>
                                </Grid>
                            </Grid>
                        );
                    })}
            </Grid>
            <Grid container>
                {isReady && (
                    <Grid xs={12}>
                        <form onSubmit={onSubmit}>
                            <Stack direction={'row'}>
                                <TextField
                                    disabled={sending}
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
                )}
            </Grid>
        </ContainerWithHeader>
    );
};
