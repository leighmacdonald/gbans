import { useCallback, useMemo, useState } from 'react';
import NiceModal, { useModal } from '@ebay/nice-modal-react';
import AccessTimeIcon from '@mui/icons-material/AccessTime';
import ConstructionIcon from '@mui/icons-material/Construction';
import LockIcon from '@mui/icons-material/Lock';
import Person2Icon from '@mui/icons-material/Person2';
import { IconButton, Theme } from '@mui/material';
import Box from '@mui/material/Box';
import Paper from '@mui/material/Paper';
import Stack from '@mui/material/Stack';
import Typography from '@mui/material/Typography';
import Grid from '@mui/material/Unstable_Grid2';
import { useTheme } from '@mui/material/styles';
import { useForm } from '@tanstack/react-form';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { createFileRoute, useNavigate, useRouteContext } from '@tanstack/react-router';
import { z } from 'zod';
import { PermissionLevel } from '../api';
import {
    apiCreateThreadReply,
    apiDeleteMessage,
    apiGetThread,
    apiGetThreadMessages,
    ForumMessage,
    ForumThread
} from '../api/forum.ts';
import { ThreadMessageContainer } from '../component/ForumThreadMessageContainer.tsx';
import { LoadingPlaceholder } from '../component/LoadingPlaceholder.tsx';
import { PaginatorLocal } from '../component/PaginatorLocal.tsx';
import RouterLink from '../component/RouterLink.tsx';
import { Title } from '../component/Title';
import { VCenterBox } from '../component/VCenterBox.tsx';
import { Buttons } from '../component/field/Buttons.tsx';
import { MarkdownField } from '../component/field/MarkdownField.tsx';
import { ModalConfirm, ModalForumThreadEditor } from '../component/modal';
import { useUserFlashCtx } from '../hooks/useUserFlashCtx.ts';
import { logErr } from '../util/errors.ts';
import { useScrollToLocation } from '../util/history.ts';
import { commonTableSearchSchema, RowsPerPage } from '../util/table.ts';
import { renderDateTime } from '../util/text.tsx';
import { LoginPage } from './_guest.login.index.tsx';

const forumThreadSearchSchema = z.object({
    ...commonTableSearchSchema
});

export const Route = createFileRoute('/_auth/forums/thread/$forum_thread_id')({
    component: ForumThreadPage,
    validateSearch: (search) => forumThreadSearchSchema.parse(search)
});

function ForumThreadPage() {
    const { forum_thread_id } = Route.useParams();
    const { page } = Route.useSearch();
    const { sendFlash } = useUserFlashCtx();
    const queryClient = useQueryClient();
    const confirmModal = useModal(ModalConfirm);
    const navigate = useNavigate();
    const theme = useTheme();

    const { data: thread, isLoading: isLoadingThread } = useQuery({
        queryKey: ['forumThread', { forum_thread_id: Number(forum_thread_id) }],
        queryFn: async () => {
            return await apiGetThread(Number(forum_thread_id));
        }
    });

    const { data: messages, isLoading: isLoadingMessages } = useQuery({
        queryKey: ['threadMessages', { forum_thread_id }],
        queryFn: async () => {
            return await apiGetThreadMessages({ forum_thread_id: Number(forum_thread_id) });
        },
        enabled: !isLoadingThread && Boolean(thread)
    });

    const [pagination, setPagination] = useState({
        pageIndex: 0, //initial page index
        pageSize: RowsPerPage.TwentyFive //default page size
    });

    const { hasPermission, permissionLevel } = useRouteContext({ from: '/_auth/forums/thread/$forum_thread_id' });

    useScrollToLocation();

    const firstPostID = useMemo(() => {
        if (Number(page) > 1 || !messages) {
            return -1;
        }
        if (messages.length > 0) {
            return messages[0].forum_message_id;
        }
        return -1;
    }, [messages, page]);

    const onEditThread = useCallback(async () => {
        try {
            const newThread = await NiceModal.show<ForumThread>(ModalForumThreadEditor, {
                thread: thread
            });

            if (newThread.forum_thread_id > 0) {
                queryClient.setQueryData(['forumThread', { forum_thread_id: Number(forum_thread_id) }], newThread);
            } else {
                await navigate({ to: '/forums' });
            }
        } catch (e) {
            logErr(e);
        }
    }, [forum_thread_id, navigate, queryClient, thread]);

    const deleteMessageMutation = useMutation({
        mutationFn: async ({ message }: { message: ForumMessage }) => {
            await apiDeleteMessage(message.forum_message_id);
        },
        onSuccess: async (_, variables) => {
            const newMessages = (messages ?? []).filter(
                (m) => m.forum_message_id != variables.message.forum_message_id
            );
            queryClient.setQueryData(['threadMessages', { forum_thread_id }], newMessages);
            sendFlash('success', `Messages deleted successfully: #${variables.message.forum_message_id}`);
            if (firstPostID == variables.message.forum_message_id) {
                await navigate({ to: '/forums' });
            }
        },
        onError: (error) => {
            sendFlash('error', `Failed to delete message: ${error}`);
        }
    });

    const onMessageDeleted = useCallback(
        async (message: ForumMessage) => {
            const isFirstMessage = firstPostID == message.forum_message_id;
            const confirmed = await confirmModal.show({
                title: 'Delete Post?',
                children: (
                    <Box>
                        {isFirstMessage && (
                            <Typography variant={'body1'} fontWeight={700} color={theme.palette.error.dark}>
                                Please be aware that by deleting the first post in the thread, this will result in the
                                deletion of the <i>entire thread</i>.
                            </Typography>
                        )}
                        <Typography variant={'body1'}>This action cannot be undone.</Typography>
                    </Box>
                )
            });

            if (!confirmed) {
                return;
            }

            deleteMessageMutation.mutate({ message });
        },
        [confirmModal, deleteMessageMutation, firstPostID, theme.palette.error.dark]
    );

    const createMessageMutation = useMutation({
        mutationFn: async ({ body_md }: { body_md: string }) => {
            return await apiCreateThreadReply(Number(forum_thread_id), body_md);
        },
        onSuccess: (message) => {
            const newMessages = [...(messages ?? []), message];
            queryClient.setQueryData(['threadMessages', { forum_thread_id }], newMessages);
            reset();
            sendFlash('success', `New message (#${message.forum_message_id} posted`);
        }
    });

    const { Field, Subscribe, handleSubmit, reset } = useForm({
        onSubmit: async ({ value }) => {
            createMessageMutation.mutate(value);
        },
        defaultValues: {
            body_md: ''
        }
    });

    const replyContainer = useMemo(() => {
        if (permissionLevel() == PermissionLevel.Guest) {
            return <LoginPage />;
        } else if (thread?.forum_thread_id && !thread?.locked) {
            return (
                <Paper>
                    <Box padding={2}>
                        <form
                            onSubmit={async (e) => {
                                e.preventDefault();
                                e.stopPropagation();
                                await handleSubmit();
                            }}
                        >
                            <Grid container spacing={2} justifyItems={'flex-end'}>
                                <Grid xs={12}>
                                    <Field
                                        name={'body_md'}
                                        children={(props) => {
                                            return (
                                                <MarkdownField
                                                    {...props}
                                                    label={'Message'}
                                                    fullwidth={true}
                                                    minHeight={400}
                                                />
                                            );
                                        }}
                                    />
                                </Grid>
                                <Grid xs={4}>
                                    <Subscribe
                                        selector={(state) => [state.canSubmit, state.isSubmitting]}
                                        children={([canSubmit, isSubmitting]) => (
                                            <Buttons
                                                reset={reset}
                                                canSubmit={canSubmit}
                                                isSubmitting={isSubmitting}
                                                submitLabel={'Reply'}
                                            />
                                        )}
                                    />
                                </Grid>
                            </Grid>
                        </form>
                    </Box>
                </Paper>
            );
        } else {
            return <></>;
        }
    }, [permissionLevel, thread?.forum_thread_id, thread?.locked, Field, Subscribe, handleSubmit, reset]);

    return (
        <Stack spacing={1}>
            {thread?.title ? <Title>{thread?.title}</Title> : null}

            <Stack direction={'row'}>
                {hasPermission(PermissionLevel.Moderator) && (
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
                        color={(theme: Theme) => theme.palette.text.primary}
                        to={`/profile/${thread?.source_id}`}
                    >
                        {thread?.personaname ?? ''}
                    </Typography>
                </VCenterBox>
                <AccessTimeIcon />
                <VCenterBox>
                    <Typography variant={'body2'}>{renderDateTime(thread?.created_on ?? new Date())}</Typography>
                </VCenterBox>
            </Stack>
            {isLoadingMessages ? (
                <LoadingPlaceholder />
            ) : (
                (messages ?? []).map((m) => (
                    <ThreadMessageContainer
                        message={m}
                        key={`thread-message-id-${m.forum_message_id}`}
                        onDelete={onMessageDeleted}
                        isFirstMessage={firstPostID == m.forum_message_id}
                    />
                ))
            )}

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
                count={(messages ?? []).length}
                rows={pagination.pageSize}
                page={pagination.pageIndex}
            />
            {thread?.locked && (
                <Paper>
                    <Typography variant={'h4'} textAlign={'center'} padding={1}>
                        <LockIcon /> Thread Locked
                    </Typography>
                </Paper>
            )}
            {replyContainer}
        </Stack>
    );
}
