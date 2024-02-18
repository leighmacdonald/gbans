import { useCallback, useMemo, useState } from 'react';
import { Link as RouterLink, useNavigate, useParams } from 'react-router-dom';
import NiceModal, { useModal } from '@ebay/nice-modal-react';
import BuildIcon from '@mui/icons-material/Build';
import LockIcon from '@mui/icons-material/Lock';
import MessageIcon from '@mui/icons-material/Message';
import PostAddIcon from '@mui/icons-material/PostAdd';
import PushPinIcon from '@mui/icons-material/PushPin';
import Avatar from '@mui/material/Avatar';
import Button from '@mui/material/Button';
import ButtonGroup from '@mui/material/ButtonGroup';
import Pagination from '@mui/material/Pagination';
import Stack from '@mui/material/Stack';
import Typography from '@mui/material/Typography';
import Grid from '@mui/material/Unstable_Grid2';
import { ErrorCode, PermissionLevel } from '../api';
import { Forum, ForumThread } from '../api/forum';
import { ContainerWithHeaderAndButtons } from '../component/ContainerWithHeaderAndButtons';
import { ForumRowLink } from '../component/ForumRowLink';
import { VCenteredElement } from '../component/Heading';
import { PermissionDenied } from '../component/PermissionDenied';
import { VCenterBox } from '../component/VCenterBox';
import {
    ModalForumForumEditor,
    ModalForumThreadCreator
} from '../component/modal';
import { useCurrentUserCtx } from '../hooks/useCurrentUserCtx.ts';
import { useForum } from '../hooks/useForum';
import { useThreads } from '../hooks/useThreads';
import { useUserFlashCtx } from '../hooks/useUserFlashCtx.ts';
import { logErr } from '../util/errors';
import { RowsPerPage } from '../util/table.ts';
import { avatarHashToURL, renderDateTime } from '../util/text.tsx';

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
                                src={avatarHashToURL(
                                    thread.avatarhash,
                                    'medium'
                                )}
                            />
                        }
                    />
                    <Stack>
                        <Stack direction={'row'} justifyContent="space-between">
                            <ForumRowLink
                                label={thread.title}
                                to={`/forums/thread/${thread.forum_thread_id}`}
                            />
                        </Stack>
                        <Stack direction={'row'} spacing={1}>
                            {thread.sticky && (
                                <VCenterBox>
                                    <PushPinIcon fontSize={'small'} />
                                </VCenterBox>
                            )}
                            {thread.locked && (
                                <VCenterBox>
                                    <LockIcon fontSize={'small'} />
                                </VCenterBox>
                            )}
                            <Typography
                                variant={'body2'}
                                component={RouterLink}
                                to={`/profile/${thread.source_id}`}
                                color={(theme) => theme.palette.text.secondary}
                                sx={{
                                    textDecoration: 'none',
                                    '&:hover': { textDecoration: 'underline' }
                                }}
                            >
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
                    <Stack direction={'row'} justifyContent={'end'} spacing={1}>
                        <Stack>
                            <Typography
                                variant={'body2'}
                                align={'right'}
                                fontWeight={700}
                                sx={{ textDecoration: 'none' }}
                                color={(theme) => theme.palette.text.primary}
                                component={RouterLink}
                                to={`/forums/thread/${thread.forum_thread_id}#${thread.recent_forum_message_id}`}
                            >
                                {renderDateTime(thread.recent_created_on)}
                            </Typography>
                            <Typography
                                align={'right'}
                                color={(theme) => theme.palette.text.secondary}
                                variant={'body2'}
                                sx={{ textDecoration: 'none' }}
                                component={RouterLink}
                                to={`/profile/${thread.recent_steam_id}`}
                            >
                                {thread.recent_personaname}
                            </Typography>
                        </Stack>
                        <VCenterBox>
                            <Avatar
                                sx={{ height: '32px', width: '32px' }}
                                alt={avatarHashToURL(
                                    thread.recent_avatarhash,
                                    'small'
                                )}
                            />
                        </VCenterBox>
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
    const [page, setPage] = useState(1);
    const modalCreate = useModal(ModalForumThreadCreator);
    const { currentUser } = useCurrentUserCtx();
    const { sendFlash } = useUserFlashCtx();
    const navigate = useNavigate();
    const id = parseInt(forum_id as string);
    const rpp = RowsPerPage.TwentyFive;
    const [forumUpdated, setForumUpdated] = useState<Forum>();

    const { data: forum, loading, error } = useForum(id);
    const { data: threads, count } = useThreads({
        forum_id: id,
        offset: (page - 1) * rpp,
        limit: rpp,
        order_by: 'updated_on',
        desc: true
    });

    const showLogin = useMemo(() => {
        if (forum && currentUser.permission_level < forum?.permission_level) {
            return true;
        }
        return !!(
            error &&
            (error.code == ErrorCode.PermissionDenied ||
                error.code == ErrorCode.LoginRequired)
        );
    }, [forum, currentUser.permission_level, error]);

    const currentForum = useMemo(() => {
        return forumUpdated ?? forum;
    }, [forum, forumUpdated]);

    const onNewThread = useCallback(async () => {
        try {
            const thread = (await modalCreate.show({
                forum_id: id
            })) as ForumThread;
            navigate(`/forums/thread/${thread.forum_thread_id}`);
            await modalCreate.hide();
        } catch (e) {
            sendFlash('error', `${e}`);
        }
    }, [id, modalCreate, navigate, sendFlash]);

    const onEditForum = useCallback(async () => {
        try {
            const forum = await NiceModal.show<Forum>(ModalForumForumEditor, {
                initial_forum_id: id
            });
            setForumUpdated(forum);
        } catch (e) {
            logErr(e);
        }
    }, [id]);

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
                    onClick={onEditForum}
                >
                    Edit
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
        return [
            <ButtonGroup key={'forum-header-buttons'}>{buttons}</ButtonGroup>
        ];
    }, [currentUser.permission_level, onEditForum, onNewThread]);

    if (showLogin && error) {
        return <PermissionDenied error={error} />;
    }

    return (
        <ContainerWithHeaderAndButtons
            title={loading ? 'Loading...' : currentForum?.title ?? 'Forum'}
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
                <Pagination
                    count={count > 0 ? Math.ceil(count / rpp) : 0}
                    page={page}
                    onChange={(_, newPage) => {
                        setPage(newPage);
                    }}
                />
            </Stack>
        </ContainerWithHeaderAndButtons>
    );
};

export default ForumPage;
