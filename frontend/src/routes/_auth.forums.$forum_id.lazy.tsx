import { useCallback, useMemo, useState } from 'react';
import NiceModal, { useModal } from '@ebay/nice-modal-react';
import BuildIcon from '@mui/icons-material/Build';
import LockIcon from '@mui/icons-material/Lock';
import MessageIcon from '@mui/icons-material/Message';
import PostAddIcon from '@mui/icons-material/PostAdd';
import PushPinIcon from '@mui/icons-material/PushPin';
import Avatar from '@mui/material/Avatar';
import Button from '@mui/material/Button';
import ButtonGroup from '@mui/material/ButtonGroup';
import Stack from '@mui/material/Stack';
import Typography from '@mui/material/Typography';
import Grid from '@mui/material/Unstable_Grid2';
import { FetchQueryOptions, useQueryClient } from '@tanstack/react-query';
import { createFileRoute, useNavigate, useRouteContext } from '@tanstack/react-router';
import { PermissionLevel } from '../api';
import { apiForum, apiGetThreads, Forum, ForumThread } from '../api/forum.ts';
import { ContainerWithHeaderAndButtons } from '../component/ContainerWithHeaderAndButtons.tsx';
import { ErrorDetails } from '../component/ErrorDetails.tsx';
import { ForumRowLink } from '../component/ForumRowLink.tsx';
import { VCenteredElement } from '../component/Heading.tsx';
import { PaginatorLocal } from '../component/PaginatorLocal.tsx';
import RouterLink from '../component/RouterLink.tsx';
import { VCenterBox } from '../component/VCenterBox.tsx';
import { ModalForumForumEditor, ModalForumThreadCreator } from '../component/modal';
import { AppError } from '../error.tsx';
import { useUserFlashCtx } from '../hooks/useUserFlashCtx.ts';
import { logErr } from '../util/errors.ts';
import { RowsPerPage } from '../util/table.ts';
import { avatarHashToURL, renderDateTime } from '../util/text.tsx';

const forumQueryKey = (forum_id: string | number) => {
    return ['forum', { forum_id: String(forum_id) }];
};

const forumThreadsQueryKey = (forum_id: string | number) => {
    return ['forumThreads', { forum_id: String(forum_id) }];
};

export const Route = createFileRoute('/_auth/forums/$forum_id')({
    component: ForumPage,
    loader: async ({ context, params }) => {
        const { forum_id } = params;
        const forumQueryOpts = {
            queryKey: forumQueryKey(forum_id),
            queryFn: async () => {
                return await apiForum(Number(forum_id));
            }
        };
        const forum = await context.queryClient.fetchQuery(forumQueryOpts);
        const threadsQueryOpts: FetchQueryOptions<ForumThread[]> = {
            queryKey: forumThreadsQueryKey(forum_id),
            queryFn: async () => {
                return (await apiGetThreads({ forum_id: Number(forum_id) })) ?? [];
            }
        };

        const threads = await context.queryClient.fetchQuery(threadsQueryOpts);
        return { forum, threads };
    },
    errorComponent: ({ error }) => {
        if (error instanceof AppError) {
            return <ErrorDetails error={error} />;
        }
        return <div>Oops</div>;
    }
});

function ForumPage() {
    const queryClient = useQueryClient();
    const { forum_id } = Route.useParams();
    const { forum, threads } = Route.useLoaderData();
    const modalCreate = useModal(ModalForumThreadCreator);
    const { hasPermission } = useRouteContext({ from: '/_auth/forums/$forum_id' });
    const { sendFlash } = useUserFlashCtx();
    const navigate = useNavigate();

    const [pagination, setPagination] = useState({
        pageIndex: 0, //initial page index
        pageSize: RowsPerPage.TwentyFive //default page size
    });

    const onNewThread = useCallback(async () => {
        try {
            const thread = (await modalCreate.show({ forum })) as ForumThread;
            await navigate({ to: `/forums/thread/${thread.forum_thread_id}` });
            await modalCreate.hide();
        } catch (e) {
            sendFlash('error', `${e}`);
        }
    }, [forum, modalCreate, navigate, sendFlash]);

    const onEditForum = useCallback(async () => {
        try {
            const editedForum = await NiceModal.show<Forum>(ModalForumForumEditor, {
                forum
            });
            queryClient.setQueryData(forumQueryKey(forum_id), editedForum);
        } catch (e) {
            logErr(e);
        }
    }, [forum, forum_id, queryClient]);

    const headerButtons = useMemo(() => {
        const buttons = [];

        if (hasPermission(PermissionLevel.Moderator)) {
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
                disabled={!hasPermission(PermissionLevel.Guest)}
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
        return [<ButtonGroup key={'forum-header-buttons'}>{buttons}</ButtonGroup>];
    }, [hasPermission, onEditForum, onNewThread]);

    return (
        <ContainerWithHeaderAndButtons title={forum.title} iconLeft={<MessageIcon />} buttons={headerButtons}>
            <Stack spacing={2}>
                {threads.map((t) => {
                    return <ForumThreadRow thread={t} key={`ft-${t.forum_thread_id}`} />;
                })}
                <PaginatorLocal
                    onRowsChange={(rows) => {
                        setPagination((prev) => {
                            return { ...prev, pageSize: rows };
                        });
                    }}
                    onPageChange={(page) => {
                        setPagination((prev) => {
                            return { ...prev, pageIndex: page };
                        });
                    }}
                    count={threads.length}
                    rows={pagination.pageSize}
                    page={pagination.pageIndex}
                />
            </Stack>
        </ContainerWithHeaderAndButtons>
    );
}

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
                        icon={<Avatar alt={thread.personaname} src={avatarHashToURL(thread.avatarhash, 'medium')} />}
                    />
                    <Stack>
                        <Stack direction={'row'} justifyContent="space-between">
                            <ForumRowLink label={thread.title} to={`/forums/thread/${thread.forum_thread_id}`} />
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
                {thread.recent_forum_message_id && thread.recent_forum_message_id > 0 ? (
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
                                alt={avatarHashToURL(thread.recent_avatarhash, 'small')}
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
