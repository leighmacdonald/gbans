import React, { useCallback, useEffect, useMemo, useState } from 'react';
import { useNavigate, useParams } from 'react-router-dom';
import NiceModal, { useModal } from '@ebay/nice-modal-react';
import BuildIcon from '@mui/icons-material/Build';
import MessageIcon from '@mui/icons-material/Message';
import PostAddIcon from '@mui/icons-material/PostAdd';
import { TablePagination } from '@mui/material';
import Avatar from '@mui/material/Avatar';
import Button from '@mui/material/Button';
import ButtonGroup from '@mui/material/ButtonGroup';
import Stack from '@mui/material/Stack';
import Typography from '@mui/material/Typography';
import Grid from '@mui/material/Unstable_Grid2';
import { defaultAvatarHash, PermissionLevel } from '../api';
import { apiForum, apiGetThreads, Forum, ForumThread } from '../api/forum';
import { ContainerWithHeaderAndButtons } from '../component/ContainerWithHeaderAndButtons';
import { ForumRowLink } from '../component/ForumRowLink';
import { VCenteredElement } from '../component/Heading';
import { RowsPerPage } from '../component/LazyTable';
import { ModalForumThreadEditor } from '../component/modal';
import { useCurrentUserCtx } from '../contexts/CurrentUserCtx';
import { useUserFlashCtx } from '../contexts/UserFlashCtx';
import { logErr } from '../util/errors';
import { renderDateTime } from '../util/text';

const ForumThreadRow = ({ thread }: { thread: ForumThread }) => {
    return (
        <Grid
            container
            spacing={1}
            sx={{
                '&:hover': {
                    backgroundColor: (theme) => theme.palette.background.default
                }
            }}
        >
            <Grid md={8} xs={12}>
                <Stack direction={'row'} spacing={2}>
                    <VCenteredElement
                        icon={
                            <Avatar
                                alt={thread.personaname}
                                src={`https://avatars.akamai.steamstatic.com/${
                                    thread.avatarhash ?? defaultAvatarHash
                                }.jpg`}
                            />
                        }
                    />
                    <Stack>
                        <ForumRowLink
                            label={thread.title}
                            to={`/forums/thread/${thread.forum_thread_id}`}
                        />

                        <Stack direction={'row'}>
                            <Typography variant={'body2'}>
                                {thread.personaname}
                            </Typography>
                        </Stack>
                    </Stack>
                </Stack>
            </Grid>
            <Grid md={1} xs={6}>
                <Grid container justifyContent="space-between">
                    <Grid xs={6}>
                        <Typography variant={'body1'} align={'left'}>
                            Replies:
                        </Typography>
                    </Grid>
                    <Grid xs={6} alignContent={'flex-end'}>
                        <Typography variant={'body1'} align={'right'}>
                            {thread.replies}
                        </Typography>
                    </Grid>
                    <Grid xs={6}>
                        <Typography variant={'body2'}>Views:</Typography>
                    </Grid>
                    <Grid xs={6} alignContent={'flex-end'}>
                        <Typography variant={'body2'} align={'right'}>
                            {thread.views}
                        </Typography>
                    </Grid>
                </Grid>
            </Grid>
            <Grid md={3} xs={6}>
                {thread.recent_forum_message_id &&
                thread.recent_forum_message_id > 0 ? (
                    <Stack direction={'row'}>
                        <Stack>
                            <Typography variant={'body1'}>
                                {renderDateTime(thread.recent_created_on)}
                            </Typography>
                            <Typography variant={'body1'}>
                                {thread.recent_personaname}
                            </Typography>
                        </Stack>
                        <VCenteredElement
                            icon={
                                <Avatar
                                    alt={thread.recent_personaname}
                                    src={`https://avatars.akamai.steamstatic.com/${
                                        thread.recent_avatarhash ??
                                        defaultAvatarHash
                                    }.jpg`}
                                />
                            }
                        />
                    </Stack>
                ) : (
                    <></>
                )}
            </Grid>
        </Grid>
    );
};

export const ForumPage = () => {
    const { forum_id } = useParams();
    const [forum, setForum] = useState<Forum>();
    const [threads, setThreads] = useState<ForumThread[]>();
    const [count, setCount] = useState(0);
    const [loading, setLoading] = useState(false);
    const [page, setPage] = useState(0);
    const [rowsPerPage, setRowsPerPage] = useState<RowsPerPage>(
        RowsPerPage.TwentyFive
    );
    const modal = useModal(ModalForumThreadEditor);
    const { currentUser } = useCurrentUserCtx();
    const { sendFlash } = useUserFlashCtx();
    const navigate = useNavigate();
    const id = parseInt(forum_id as string);

    useEffect(() => {
        setLoading(true);
        const abortController = new AbortController();

        apiForum(id, abortController)
            .then((f) => {
                setForum(f);
            })
            .catch((e) => {
                logErr(e);
            })
            .finally(() => {
                setLoading(false);
            });

        return () => abortController.abort();
    }, [id]);

    useEffect(() => {
        apiGetThreads({
            forum_id: id,
            offset: page * rowsPerPage,
            limit: rowsPerPage,
            order_by: 'updated_on',
            desc: true
        }).then((resp) => {
            setThreads(resp.data);
            setCount(resp.count);
        });
    }, [id, page, rowsPerPage]);

    const onNewThread = useCallback(async () => {
        try {
            const thread = await NiceModal.show<ForumThread>(
                ModalForumThreadEditor,
                { forum_id: id }
            );
            navigate(`/forums/thread/${thread.forum_thread_id}`);
            await modal.hide();
        } catch (e) {
            sendFlash('error', `${e}`);
        }
    }, [id, modal, navigate, sendFlash]);

    const headerButtons = useMemo(() => {
        const buttons = [];

        if (currentUser.permission_level >= PermissionLevel.Moderator) {
            buttons.push(
                <Button
                    startIcon={<BuildIcon />}
                    color={'warning'}
                    variant={'contained'}
                    size={'small'}
                    key={'btn-edit-forum'}
                >
                    Edit Forum
                </Button>
            );
        }
        buttons.push(
            <Button
                disabled={currentUser.permission_level <= PermissionLevel.Guest}
                variant={'contained'}
                color={'success'}
                size={'small'}
                onClick={onNewThread}
                startIcon={<PostAddIcon />}
                key={'btn-new-post'}
            >
                New Post
            </Button>
        );
        return [<ButtonGroup key={'forum-header-btns'}>{buttons}</ButtonGroup>];
    }, [currentUser.permission_level, onNewThread]);

    return (
        <ContainerWithHeaderAndButtons
            title={loading ? 'Loading...' : forum?.title ?? 'Forum'}
            iconLeft={<MessageIcon />}
            buttons={headerButtons}
        >
            <Stack spacing={2}>
                {threads?.map((t) => {
                    return (
                        <ForumThreadRow
                            thread={t}
                            key={`ft-${t.forum_thread_id}`}
                        />
                    );
                })}
                <TablePagination
                    component={'div'} // Stops error since this is not a real <Table>
                    rowsPerPage={rowsPerPage}
                    count={count}
                    page={page}
                    onRowsPerPageChange={(event) => {
                        setRowsPerPage(parseInt(event.target.value));
                    }}
                    onPageChange={(_, newPage) => {
                        setPage(newPage);
                    }}
                />
            </Stack>
        </ContainerWithHeaderAndButtons>
    );
};
