import Grid from '@mui/material/Grid/Grid';
import React, { FC, useCallback, useEffect, useRef, useState } from 'react';
import Typography from '@mui/material/Typography';
import Stack from '@mui/material/Stack';
import TextField from '@mui/material/TextField';
import Button from '@mui/material/Button';
import { ContainerWithHeader } from './ContainerWithHeader';
import { wsMsgTypePugUserMessageResponse } from '../pug/pug';
import { parseDateTime, renderTime } from '../util/text';

export const UserMessagesView = ({
    messages
}: {
    messages: wsMsgTypePugUserMessageResponse[];
}) => {
    const messagesEndRef = useRef<HTMLDivElement>(null);

    const scrollToBottom = () => {
        messagesEndRef.current?.scrollIntoView({ behavior: 'smooth' });
    };
    useEffect(() => {
        scrollToBottom();
    }, [messages]);

    return (
        <Stack height={300} padding={1} spacing={1} overflow={'scroll'}>
            {messages.map((m, i) => {
                return (
                    <Stack direction={'row'} spacing={2} key={`msg-${i}`}>
                        <Typography variant={'body2'}>
                            {renderTime(parseDateTime(m.created_at))}
                        </Typography>
                        <Typography variant={'body2'}>
                            {m.user ? m.user.name : 'Lobby'}
                        </Typography>
                        <Typography variant={'body2'}>{m.message}</Typography>
                    </Stack>
                );
            })}
            <div ref={messagesEndRef} />
        </Stack>
    );
};

export interface ChatViewProps {
    messages: wsMsgTypePugUserMessageResponse[];
    sendMessage: (message: string) => void;
}

export const ChatView: FC<ChatViewProps> = ({ messages, sendMessage }) => {
    const [msg, setMsg] = useState('');
    const onSend = useCallback(() => {
        if (msg == '') {
            return;
        }
        sendMessage(msg);
        setMsg('');
    }, [msg, sendMessage]);

    return (
        <ContainerWithHeader title={'Lobby Chat'}>
            <Grid item>
                <Grid item xs={12}>
                    <UserMessagesView messages={messages} />
                </Grid>
                <Grid item xs={12}>
                    <Stack direction={'row'} padding={1} spacing={1}>
                        <TextField
                            fullWidth
                            size={'small'}
                            id="outlined-basic"
                            variant="standard"
                            value={msg}
                            onKeyDown={(evt) => {
                                if (evt.key !== 'Enter') {
                                    return;
                                }
                                onSend();
                            }}
                            onChange={(event) => {
                                setMsg(event.target.value);
                            }}
                        />
                        <Button
                            size={'small'}
                            variant={'text'}
                            color={'primary'}
                            onClick={onSend}
                            disabled={msg.length === 0}
                        >
                            Send
                        </Button>
                    </Stack>
                </Grid>
            </Grid>
        </ContainerWithHeader>
    );
};
