import { useMemo, useState } from 'react';
import DeleteForeverIcon from '@mui/icons-material/DeleteForever';
import EditIcon from '@mui/icons-material/Edit';
import { Divider, IconButton, Theme } from '@mui/material';
import Box from '@mui/material/Box';
import Paper from '@mui/material/Paper';
import Stack from '@mui/material/Stack';
import Typography from '@mui/material/Typography';
import Grid from '@mui/material/Unstable_Grid2';
import { useTheme } from '@mui/material/styles';
import { useForm } from '@tanstack/react-form';
import { useMutation } from '@tanstack/react-query';
import { useRouteContext } from '@tanstack/react-router';
import { zodValidator } from '@tanstack/zod-form-adapter';
import { isAfter } from 'date-fns/fp';
import { z } from 'zod';
import { PermissionLevel, permissionLevelString } from '../api';
import { apiSaveThreadMessage, ForumMessage } from '../api/forum.ts';
import { useUserFlashCtx } from '../hooks/useUserFlashCtx.ts';
import { avatarHashToURL } from '../util/text.tsx';
import { renderDateTime } from '../util/time.ts';
import { ForumAvatar } from './ForumAvatar.tsx';
import { ForumRowLink } from './ForumRowLink.tsx';
import { MarkDownRenderer } from './MarkdownRenderer.tsx';
import RouterLink from './RouterLink.tsx';
import { Buttons } from './field/Buttons.tsx';
import { MarkdownField, mdEditorRef } from './field/MarkdownField.tsx';

export const ThreadMessageContainer = ({
    message,
    onDelete,
    onSave
}: {
    message: ForumMessage;
    onDelete: (message: ForumMessage) => Promise<void>;
    onSave: (message: ForumMessage) => Promise<void>;
    isFirstMessage: boolean;
}) => {
    const [edit, setEdit] = useState(false);
    const { hasPermission, profile } = useRouteContext({ from: '/_auth/forums/thread/$forum_thread_id' });
    const { sendError } = useUserFlashCtx();
    const theme = useTheme();

    const editable = useMemo(() => {
        return profile.steam_id == message.source_id || hasPermission(PermissionLevel.Moderator);
    }, [hasPermission, message.source_id, profile.steam_id]);

    const mutation = useMutation({
        mutationFn: async (variables: { body_md: string }) => {
            return await apiSaveThreadMessage(message.forum_message_id, variables.body_md);
        },
        onSuccess: async (data) => {
            mdEditorRef.current?.setMarkdown('');
            setEdit(false);
            await onSave(data);
        },
        onError: sendError
    });

    const form = useForm({
        onSubmit: async ({ value }) => {
            mutation.mutate({
                body_md: value.body_md ?? ''
            });
        },
        validatorAdapter: zodValidator,
        defaultValues: {
            body_md: message.body_md
        }
    });

    return (
        <Paper elevation={1} id={`${message.forum_message_id}`}>
            <Grid container>
                <Grid xs={2} padding={2} sx={{ backgroundColor: theme.palette.background.paper }}>
                    <Stack alignItems={'center'}>
                        <ForumAvatar
                            alt={message.personaname}
                            online={message.online}
                            src={avatarHashToURL(message.avatarhash, 'medium')}
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
                    {edit ? (
                        <form
                            onSubmit={async (e) => {
                                e.preventDefault();
                                e.stopPropagation();
                                await form.handleSubmit();
                            }}
                        >
                            <Stack padding={1}>
                                <form.Field
                                    name={'body_md'}
                                    validators={{
                                        onChange: z.string().min(4)
                                    }}
                                    children={(props) => {
                                        return <MarkdownField {...props} label={'Message (Markdown)'} />;
                                    }}
                                />
                                <form.Subscribe
                                    selector={(state) => [state.canSubmit, state.isSubmitting]}
                                    children={([canSubmit, isSubmitting]) => (
                                        <Buttons canSubmit={canSubmit} isSubmitting={isSubmitting} reset={form.reset} />
                                    )}
                                />
                            </Stack>
                        </form>
                    ) : (
                        <Box>
                            <Grid container direction="row" borderBottom={(theme) => theme.palette.divider}>
                                <Grid xs={6}>
                                    <Stack direction={'row'}>
                                        <Typography variant={'body2'} padding={1}>
                                            {renderDateTime(message.created_on)}
                                        </Typography>
                                        {isAfter(message.created_on, message.updated_on) && (
                                            <Typography variant={'body2'} padding={1}>
                                                {`Edited: ${renderDateTime(message.updated_on)}`}
                                            </Typography>
                                        )}
                                    </Stack>
                                </Grid>
                                <Grid xs={6}>
                                    <Stack direction="row" justifyContent="end">
                                        <IconButton
                                            color={'error'}
                                            onClick={async () => {
                                                await onDelete(message);
                                            }}
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
                                            to={`#${message.forum_message_id}`}
                                            textAlign={'right'}
                                            color={(theme: Theme) => {
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

                                {message.signature != '' && (
                                    <>
                                        <Divider />
                                        <MarkDownRenderer body_md={message.signature} />
                                    </>
                                )}
                            </Grid>
                        </Box>
                    )}
                </Grid>
            </Grid>
        </Paper>
    );
};
