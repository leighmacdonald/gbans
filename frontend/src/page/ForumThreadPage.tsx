import React, { useCallback, useEffect, useMemo, useState, JSX } from 'react';
import { Link as RouterLink, useNavigate, useParams } from 'react-router-dom';
import NiceModal, { useModal } from '@ebay/nice-modal-react';
import AccessTimeIcon from '@mui/icons-material/AccessTime';
import ConstructionIcon from '@mui/icons-material/Construction';
import DeleteForeverIcon from '@mui/icons-material/DeleteForever';
import EditIcon from '@mui/icons-material/Edit';
import LockIcon from '@mui/icons-material/Lock';
import Person2Icon from '@mui/icons-material/Person2';
import { IconButton } from '@mui/material';
import Avatar from '@mui/material/Avatar';
import Box from '@mui/material/Box';
import Button from '@mui/material/Button';
import ButtonGroup from '@mui/material/ButtonGroup';
import Pagination from '@mui/material/Pagination';
import Paper from '@mui/material/Paper';
import Stack from '@mui/material/Stack';
import Typography from '@mui/material/Typography';
import Grid from '@mui/material/Unstable_Grid2';
import { useTheme } from '@mui/material/styles';
import { isAfter } from 'date-fns/fp';
import { Formik } from 'formik';
import * as yup from 'yup';
import { PermissionLevel, permissionLevelString } from '../api';
import {
    apiDeleteMessage,
    apiGetThread,
    apiGetThreadMessages,
    apiSaveThreadMessage,
    ForumMessage,
    ForumThread
} from '../api/forum';
import { ForumRowLink } from '../component/ForumRowLink';
import { ForumThreadReplyBox } from '../component/ForumThreadReplyBox';
import { bodyMDValidator, MDBodyField } from '../component/MDBodyField';
import { MarkDownRenderer } from '../component/MarkdownRenderer';
import { VCenterBox } from '../component/VCenterBox';
import { ModalConfirm, ModalForumThreadEditor } from '../component/modal';
import { SubmitButton } from '../component/modal/Buttons';
import { RowsPerPage } from '../component/table/LazyTable';
import { hasPermission, useCurrentUserCtx } from '../contexts/CurrentUserCtx';
import { useUserFlashCtx } from '../contexts/UserFlashCtx';
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

const ThreadMessageContainer = ({
    message,
    isFirstMessage,
    onDeleteSuccess
}: {
    message: ForumMessage;
    onDeleteSuccess: (forum_message_id: number) => void;
    isFirstMessage: boolean;
}) => {
    const [edit, setEdit] = useState(false);
    const [updatedMessage, setUpdatedMessage] = useState<ForumMessage>();
    const confirmModal = useModal(ModalConfirm);
    const { currentUser } = useCurrentUserCtx();
    const theme = useTheme();
    const navigate = useNavigate();

    const activeMessage = useMemo(() => {
        if (updatedMessage != undefined) {
            return updatedMessage;
        }
        return message;
    }, [message, updatedMessage]);

    const onUpdate = useCallback((updated: ForumMessage) => {
        setUpdatedMessage(updated);
        setEdit(false);
    }, []);

    const editable = useMemo(() => {
        return (
            currentUser.steam_id == message.source_id ||
            currentUser.permission_level >= PermissionLevel.Moderator
        );
    }, [currentUser.permission_level, currentUser.steam_id, message.source_id]);

    const onDelete = useCallback(async () => {
        try {
            const confirmed = await confirmModal.show({
                title: 'Delete Post?',
                children: (
                    <Box>
                        {isFirstMessage && (
                            <Typography
                                variant={'body1'}
                                fontWeight={700}
                                color={theme.palette.error.dark}
                            >
                                Please be aware that by deleting the first post
                                in the thread, this will result in the deletion
                                of the <i>entire thread</i>.
                            </Typography>
                        )}
                        <Typography variant={'body1'}>
                            This action cannot be undone.
                        </Typography>
                    </Box>
                )
            });
            if (confirmed) {
                await apiDeleteMessage(activeMessage.forum_message_id);
                onDeleteSuccess(activeMessage.forum_message_id);
            }
            await confirmModal.hide();
            if (isFirstMessage) {
                navigate('/forums');
            }
        } catch (e) {
            logErr(e);
        }
    }, [
        activeMessage.forum_message_id,
        confirmModal,
        isFirstMessage,
        navigate,
        onDeleteSuccess,
        theme.palette.error.dark
    ]);

    return (
        <Paper elevation={1} id={`${activeMessage.forum_message_id}`}>
            <Grid container>
                <Grid
                    xs={2}
                    padding={2}
                    sx={{ backgroundColor: theme.palette.background.paper }}
                >
                    <Stack alignItems={'center'}>
                        <ForumAvatar
                            src={`https://avatars.akamai.steamstatic.com/${activeMessage.avatarhash}_full.jpg`}
                        />
                        <ForumRowLink
                            label={activeMessage.personaname}
                            to={`/profile/${activeMessage.source_id}`}
                            align={'center'}
                        />
                        <Typography variant={'subtitle1'} align={'center'}>
                            {permissionLevelString(
                                activeMessage.permission_level
                            )}
                        </Typography>
                    </Stack>
                </Grid>
                <Grid xs={10}>
                    {edit ? (
                        <MessageEditor
                            message={activeMessage}
                            onUpdate={onUpdate}
                            onCancel={() => {
                                setEdit(false);
                            }}
                        />
                    ) : (
                        <Box>
                            <Grid
                                container
                                direction="row"
                                borderBottom={(theme) => theme.palette.divider}
                            >
                                <Grid xs={6}>
                                    <Stack direction={'row'}>
                                        <Typography
                                            variant={'body2'}
                                            padding={1}
                                        >
                                            {renderDateTime(
                                                activeMessage.created_on
                                            )}
                                        </Typography>
                                        {isAfter(
                                            message.created_on,
                                            message.updated_on
                                        ) && (
                                            <Typography
                                                variant={'body2'}
                                                padding={1}
                                            >
                                                {`Edited: ${renderDateTime(
                                                    activeMessage.updated_on
                                                )}`}
                                            </Typography>
                                        )}
                                    </Stack>
                                </Grid>
                                <Grid xs={6}>
                                    <Stack direction="row" justifyContent="end">
                                        <IconButton
                                            color={'error'}
                                            onClick={onDelete}
                                        >
                                            <DeleteForeverIcon />
                                        </IconButton>
                                        {editable && (
                                            <IconButton
                                                title={'Edit Post'}
                                                color={'secondary'}
                                                size={'small'}
                                                onClick={() => {
                                                    setEdit(true);
                                                }}
                                            >
                                                <EditIcon />
                                            </IconButton>
                                        )}
                                        <Typography
                                            padding={1}
                                            component={RouterLink}
                                            variant={'body2'}
                                            to={`#${activeMessage.forum_message_id}`}
                                            textAlign={'right'}
                                            color={(theme) => {
                                                return theme.palette.text
                                                    .primary;
                                            }}
                                        >
                                            {`#${activeMessage.forum_message_id}`}
                                        </Typography>
                                    </Stack>
                                </Grid>
                            </Grid>
                            <Grid xs={12} padding={1}>
                                <MarkDownRenderer
                                    body_md={activeMessage.body_md}
                                />
                            </Grid>
                        </Box>
                    )}
                </Grid>
            </Grid>
        </Paper>
    );
};

interface MessageEditValues {
    body_md: string;
}

const validationSchema = yup.object({
    body_md: bodyMDValidator
});

const MessageEditor = ({
    message,
    onUpdate,
    onCancel
}: {
    message: ForumMessage;
    onUpdate: (msg: ForumMessage) => void;
    onCancel: () => void;
}) => {
    const onSubmit = useCallback(
        async (values: MessageEditValues) => {
            try {
                const updated = await apiSaveThreadMessage(
                    message.forum_message_id,
                    values.body_md
                );
                onUpdate(updated);
            } catch (e) {
                logErr(e);
            }
        },
        [message.forum_message_id, onUpdate]
    );

    return (
        <Formik<MessageEditValues>
            onSubmit={onSubmit}
            initialValues={{ body_md: message.body_md }}
            validationSchema={validationSchema}
            validateOnBlur={true}
        >
            <Stack padding={1}>
                <MDBodyField />
                <ButtonGroup>
                    <Button
                        variant={'contained'}
                        color={'error'}
                        onClick={onCancel}
                    >
                        Cancel
                    </Button>
                    <SubmitButton />
                </ButtonGroup>
            </Stack>
        </Formik>
    );
};

export const ForumThreadPage = (): JSX.Element => {
    const [thread, setThread] = useState<ForumThread>();
    const [messages, setMessages] = useState<ForumMessage[]>([]);
    const [count, setCount] = useState(0);
    const [page, setPage] = useState(1);
    const { forum_thread_id } = useParams();
    const { currentUser } = useCurrentUserCtx();
    const thread_id = parseInt(forum_thread_id ?? '');
    const navigate = useNavigate();
    const { sendFlash } = useUserFlashCtx();

    useScrollToLocation();

    useEffect(() => {
        const abortController = new AbortController();
        apiGetThread(thread_id, abortController)
            .then((resp) => {
                setThread(resp);
            })
            .catch((e) => {
                logErr(e);
            });
        return () => abortController.abort();
    }, [navigate, sendFlash, thread_id]);

    useEffect(() => {
        const abortController = new AbortController();
        apiGetThreadMessages(
            {
                forum_thread_id: thread_id,
                offset: (page - 1) * RowsPerPage.Ten,
                limit: RowsPerPage.Ten,
                order_by: 'forum_message_id',
                desc: false
            },
            abortController
        ).then((m) => {
            setMessages(m.data);
            setCount(m.count);
        });
        return () => abortController.abort();
    }, [page, thread_id]);

    const firstPostID = useMemo(() => {
        if (page > 1) {
            return -1;
        }
        if (messages.length > 0) {
            return messages[0].forum_message_id;
        }
        return -1;
    }, [messages, page]);

    const isMod = useMemo(() => {
        return hasPermission(currentUser, PermissionLevel.Moderator);
    }, [currentUser]);

    const onEditThread = useCallback(async () => {
        try {
            const newThread = await NiceModal.show<ForumThread>(
                ModalForumThreadEditor,
                {
                    thread
                }
            );
            if (newThread.forum_thread_id > 0) {
                setThread(newThread);
            } else {
                navigate('/forums/');
            }
        } catch (e) {
            logErr(e);
        }
    }, [navigate, thread]);

    const onMessageDeleted = useCallback((forum_message_id: number) => {
        setMessages((prevState) => {
            return prevState.filter(
                (m) => m.forum_message_id != forum_message_id
            );
        });
    }, []);

    return (
        <Stack spacing={1}>
            <Stack direction={'row'}>
                {isMod && (
                    <IconButton color={'warning'} onClick={onEditThread}>
                        <ConstructionIcon fontSize={'small'} />
                    </IconButton>
                )}
                <Typography variant={'h3'}>{thread?.title}</Typography>
            </Stack>
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
                    key={`thread-message-id-${m.forum_message_id}`}
                    onDeleteSuccess={onMessageDeleted}
                    isFirstMessage={firstPostID == m.forum_message_id}
                />
            ))}
            <Pagination
                count={count > 0 ? Math.ceil(count / RowsPerPage.Ten) : 0}
                page={page}
                onChange={(_, newPage) => {
                    setPage(newPage);
                }}
            />
            {thread?.locked && (
                <Paper>
                    <Typography variant={'h4'} textAlign={'center'} padding={1}>
                        <LockIcon /> Thread Locked
                    </Typography>
                </Paper>
            )}
            {thread?.forum_thread_id && !thread?.locked && (
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
