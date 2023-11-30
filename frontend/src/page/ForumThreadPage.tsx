import React, { useEffect, useState } from 'react';
import { Link as RouterLink, useParams } from 'react-router-dom';
import AccessTimeIcon from '@mui/icons-material/AccessTime';
import Person2Icon from '@mui/icons-material/Person2';
import Avatar from '@mui/material/Avatar';
import Box from '@mui/material/Box';
import Pagination from '@mui/material/Pagination';
import Paper from '@mui/material/Paper';
import Stack from '@mui/material/Stack';
import Typography from '@mui/material/Typography';
import Grid from '@mui/material/Unstable_Grid2';
import { useTheme } from '@mui/material/styles';
import { permissionLevelString } from '../api';
import {
    apiGetThread,
    apiGetThreadMessages,
    ForumMessage,
    ForumThread
} from '../api/forum';
import { ForumRowLink } from '../component/ForumRowLink';
import { ForumThreadReplyBox } from '../component/ForumThreadReplyBox';
import { MarkDownRenderer } from '../component/MarkdownRenderer';
import { VCenterBox } from '../component/VCenterBox';
import { RowsPerPage } from '../component/table/LazyTable';
import { logErr } from '../util/errors';
import { useScrollToLocation } from '../util/history';
import { renderDateTime } from '../util/text';

const ForumAvatar = ({ ...props }) => (
    <Avatar
        variant={'square'}
        sx={{ height: '120px', width: '120px' }}
        {...props}
    />
);

const ThreadMessageContainer = ({ message }: { message: ForumMessage }) => {
    const theme = useTheme();
    return (
        <Paper elevation={1} id={`${message.forum_message_id}`}>
            <Grid container>
                <Grid
                    xs={2}
                    padding={2}
                    sx={{ backgroundColor: theme.palette.background.paper }}
                >
                    <Stack alignItems={'center'}>
                        <ForumAvatar
                            src={`https://avatars.akamai.steamstatic.com/${message.avatarhash}_full.jpg`}
                        />
                        <ForumRowLink
                            label={message.personaname}
                            to={`/profile/${message.source_id}`}
                            align={'center'}
                        />
                        <Typography variant={'subtitle1'} align={'center'}>
                            {permissionLevelString(message.permission_level)}
                        </Typography>
                    </Stack>
                </Grid>
                <Grid xs={10}>
                    <Box>
                        <Grid
                            container
                            direction="row"
                            borderBottom={(theme) => theme.palette.divider}
                        >
                            <Grid xs={6}>
                                <Typography variant={'body2'} padding={1}>
                                    {renderDateTime(message.created_on)}
                                </Typography>
                            </Grid>
                            <Grid xs={6}>
                                <Stack direction="row" justifyContent="end">
                                    <Typography
                                        padding={1}
                                        component={RouterLink}
                                        variant={'body2'}
                                        to={`#${message.forum_message_id}`}
                                        textAlign={'right'}
                                        color={(theme) => {
                                            return theme.palette.text.primary;
                                        }}
                                    >
                                        {`#${message.forum_message_id}`}
                                    </Typography>
                                </Stack>
                            </Grid>
                        </Grid>
                        <Grid xs={12} padding={1}>
                            <MarkDownRenderer body_md={message.body_md} />
                        </Grid>
                    </Box>
                </Grid>
            </Grid>
        </Paper>
    );
};

export const ForumThreadPage = (): JSX.Element => {
    const [thread, setThread] = useState<ForumThread>();
    const [messages, setMessages] = useState<ForumMessage[]>([]);
    const [count, setCount] = useState(0);
    const [page, setPage] = useState(1);
    const { forum_thread_id } = useParams();
    const thread_id = parseInt(forum_thread_id ?? '');

    useScrollToLocation();

    useEffect(() => {
        const abortController = new AbortController();
        apiGetThread(thread_id, abortController)
            .then((resp) => {
                setThread(resp);
            })
            .catch((e) => logErr(e));
        return () => abortController.abort();
    }, [thread_id]);

    useEffect(() => {
        const abortController = new AbortController();
        apiGetThreadMessages(
            {
                forum_thread_id: thread_id,
                offset: page * RowsPerPage.Ten,
                limit: RowsPerPage.Ten,
                order_by: 'updated_on',
                desc: true
            },
            abortController
        ).then((m) => {
            setMessages(m.data);
            setCount(m.count);
        });
        return () => abortController.abort();
    }, [page, thread_id]);

    return (
        <Stack spacing={1}>
            <Typography variant={'h3'}>{thread?.title}</Typography>
            <Stack direction={'row'} spacing={1}>
                <Person2Icon />
                <VCenterBox>
                    <Typography
                        variant={'body2'}
                        component={RouterLink}
                        color={(theme) => theme.palette.text.primary}
                        to={`/profile/${thread?.source_id}`}
                    >
                        {thread?.personaname ?? ''}
                    </Typography>
                </VCenterBox>
                <AccessTimeIcon />
                <VCenterBox>
                    <Typography variant={'body2'}>
                        {renderDateTime(thread?.created_on ?? new Date())}
                    </Typography>
                </VCenterBox>
            </Stack>
            {messages.map((m) => (
                <ThreadMessageContainer
                    message={m}
                    key={`tmid-${m.forum_message_id}`}
                />
            ))}
            <Pagination
                count={count > 0 ? Math.ceil(count / RowsPerPage.Ten) : 0}
                page={page}
                onChange={(_, newPage) => {
                    setPage(newPage);
                }}
            />

            {thread?.forum_thread_id && (
                <ForumThreadReplyBox
                    forum_thread_id={thread?.forum_thread_id}
                    onSuccess={(message) => {
                        setMessages((prevState) => {
                            return [...prevState, message];
                        });
                    }}
                />
            )}
        </Stack>
    );
};
