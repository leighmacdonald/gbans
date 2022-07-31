import React, { useCallback, useEffect, useState } from 'react';
import Grid from '@mui/material/Grid';
import { useParams } from 'react-router-dom';
import {
    apiCreateBanMessage,
    apiDeleteBanMessage,
    apiGetBan,
    apiGetBanMessages,
    apiUpdateBanMessage,
    AuthorMessage,
    BannedPerson,
    BanReasons,
    UserMessage
} from '../api';
import { NotNull } from '../util/types';
import { Heading } from '../component/Heading';
import { SteamIDList } from '../component/SteamIDList';
import { ProfileInfoBox } from '../component/ProfileInfoBox';
import Paper from '@mui/material/Paper';
import Stack from '@mui/material/Stack';
import ListItem from '@mui/material/ListItem';
import ListItemText from '@mui/material/ListItemText';
import List from '@mui/material/List';
import { MDEditor } from '../component/MDEditor';
import { UserMessageView } from '../component/UserMessageView';
import { logErr } from '../util/errors';
import { useUserFlashCtx } from '../contexts/UserFlashCtx';
import { useCurrentUserCtx } from '../contexts/CurrentUserCtx';

export const BanPage = (): JSX.Element => {
    //const [loading, setLoading] = React.useState<boolean>(true);
    const [ban, setBan] = React.useState<NotNull<BannedPerson>>();
    const [messages, setMessages] = useState<AuthorMessage[]>([]);
    const { currentUser } = useCurrentUserCtx();
    const { sendFlash } = useUserFlashCtx();
    const { ban_id_str } = useParams();
    const ban_id = parseInt(ban_id_str || '0');

    useEffect(() => {
        apiGetBan(ban_id)
            .then((banPerson) => {
                if (banPerson) {
                    setBan(banPerson);
                }
                //setLoading(false);
            })
            .catch((e) => {
                alert(`Failed to load ban: ${e}`);
            });

        // eslint-disable-next-line react-hooks/exhaustive-deps
    }, []);

    const loadMessages = useCallback(() => {
        apiGetBanMessages(ban_id)
            .then((r) => {
                setMessages(r || []);
            })
            .catch(logErr);
    }, [ban_id]);

    const onSave = useCallback(
        (message: string) => {
            apiCreateBanMessage(ban_id, message)
                .then((response) => {
                    setMessages([
                        ...messages,
                        { author: currentUser, message: response }
                    ]);
                })
                .catch(logErr);
        },
        [messages, ban_id, currentUser]
    );

    const onEdit = useCallback(
        (message: UserMessage) => {
            apiUpdateBanMessage(message.message_id, message.contents)
                .then(() => {
                    sendFlash('success', 'Updated message successfully');
                    loadMessages();
                })
                .catch(logErr);
        },
        [loadMessages, sendFlash]
    );

    const onDelete = useCallback(
        (message_id: number) => {
            apiDeleteBanMessage(message_id)
                .then(() => {
                    sendFlash('success', 'Deleted message successfully');
                    loadMessages();
                })
                .catch(logErr);
        },
        [loadMessages, sendFlash]
    );

    return (
        <Grid container paddingTop={3} spacing={3}>
            <Grid item xs={6}>
                <Stack>
                    <Heading>Comments / Appeal</Heading>
                    {messages.map((m) => (
                        <UserMessageView
                            onSave={onEdit}
                            onDelete={onDelete}
                            author={m.author}
                            message={m.message}
                            key={m.message.message_id}
                        />
                    ))}
                    <Paper elevation={1}>
                        <Stack spacing={2}>
                            <MDEditor
                                initialBodyMDValue={''}
                                onSave={onSave}
                                saveLabel={'Send Message'}
                            />
                        </Stack>
                    </Paper>
                </Stack>
            </Grid>
            <Grid item xs={6}>
                <Grid container spacing={3}>
                    <Grid item xs={12}>
                        {ban && (
                            <ProfileInfoBox
                                profile={{ player: ban?.person, friends: [] }}
                            />
                        )}
                    </Grid>
                    <Grid item xs={5}>
                        {ban && (
                            <Paper elevation={1}>
                                <SteamIDList steam_id={ban?.ban.steam_id} />
                            </Paper>
                        )}
                    </Grid>
                    <Grid item xs={7}>
                        {ban && (
                            <Paper elevation={1}>
                                <Stack>
                                    <Heading>Ban Details</Heading>
                                    <List dense={true}>
                                        <ListItem>
                                            <ListItemText
                                                primary={'Reason'}
                                                secondary={
                                                    BanReasons[ban.ban.reason]
                                                }
                                            />
                                        </ListItem>
                                        {ban.ban.reason_text != '' && (
                                            <ListItem>
                                                <ListItemText
                                                    primary={'Reason (Custom)'}
                                                    secondary={
                                                        ban.ban.reason_text
                                                    }
                                                />
                                            </ListItem>
                                        )}
                                        <ListItem>
                                            <ListItemText
                                                primary={'Created On'}
                                                secondary={
                                                    ban.ban
                                                        .created_on as any as string
                                                }
                                            />
                                        </ListItem>
                                        <ListItem>
                                            <ListItemText
                                                primary={'Expires'}
                                                secondary={
                                                    ban.ban
                                                        .valid_until as any as string
                                                }
                                            />
                                        </ListItem>
                                    </List>
                                </Stack>
                            </Paper>
                        )}
                    </Grid>
                </Grid>
            </Grid>
        </Grid>
    );
};
