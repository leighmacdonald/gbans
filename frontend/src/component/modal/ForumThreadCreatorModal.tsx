import { useCallback } from 'react';
import NiceModal, { muiDialogV5, useModal } from '@ebay/nice-modal-react';
import { Dialog, DialogActions, DialogContent, DialogTitle } from '@mui/material';
import Grid from '@mui/material/Unstable_Grid2';
import { useTheme } from '@mui/material/styles';
import useMediaQuery from '@mui/material/useMediaQuery';
import { useForm } from '@tanstack/react-form';
import { useMutation } from '@tanstack/react-query';
import { z } from 'zod';
import { apiCreateThread, Forum, ForumThread } from '../../api/forum.ts';
import { useUserFlashCtx } from '../../hooks/useUserFlashCtx.ts';
import { logErr } from '../../util/errors';
import { Buttons } from '../field/Buttons.tsx';
import { CheckboxSimple } from '../field/CheckboxSimple.tsx';
import { MarkdownField, mdEditorRef } from '../field/MarkdownField.tsx';
import { TextFieldSimple } from '../field/TextFieldSimple.tsx';
import { ModalConfirm, ModalForumThreadCreator } from './index';

type ForumThreadEditorValues = {
    title: string;
    body_md: string;
    sticky: boolean;
    locked: boolean;
};

export const ForumThreadCreatorModal = NiceModal.create(({ forum }: { forum: Forum }) => {
    const threadModal = useModal(ModalForumThreadCreator);
    const confirmModal = useModal(ModalConfirm);
    const { sendError } = useUserFlashCtx();
    const theme = useTheme();
    const modal = useModal();
    const fullScreen = useMediaQuery(theme.breakpoints.down('md'));

    const onClose = useCallback(
        async (_: unknown, reason: 'escapeKeyDown' | 'backdropClick') => {
            if (reason == 'backdropClick') {
                try {
                    const confirmed = await confirmModal.show({
                        title: 'Cancel thread creation?',
                        children: 'All progress will be lost'
                    });
                    if (confirmed) {
                        await confirmModal.hide();
                        await threadModal.hide();
                    } else {
                        await confirmModal.hide();
                    }
                } catch (e) {
                    logErr(e);
                }
            }
        },
        [confirmModal, threadModal]
    );

    const mutation = useMutation({
        mutationKey: ['forumThreadCreate', { forum_id: forum.forum_id }],
        mutationFn: async (values: ForumThreadEditorValues) => {
            return await apiCreateThread(forum.forum_id, values.title, values.body_md, values.sticky, values.locked);
        },
        onSuccess: async (editedThread: ForumThread) => {
            modal.resolve(editedThread);
            mdEditorRef.current?.setMarkdown('');
            await modal.hide();
        },
        onError: sendError
    });

    const { Field, Subscribe, handleSubmit, reset } = useForm({
        onSubmit: async ({ value }) => {
            mutation.mutate({ ...value });
        },
        validators: {
            onChange: z.object({
                title: z.string().min(3),
                body_md: z.string().min(10),
                sticky: z.boolean(),
                locked: z.boolean()
            })
        },
        defaultValues: {
            title: '',
            body_md: '',
            sticky: false,
            locked: false
        }
    });

    return (
        <Dialog
            {...muiDialogV5(threadModal)}
            fullWidth
            maxWidth={'lg'}
            closeAfterTransition={false}
            onClose={onClose}
            fullScreen={fullScreen}
        >
            <form
                onSubmit={async (e) => {
                    e.preventDefault();
                    e.stopPropagation();
                    await handleSubmit();
                }}
            >
                <DialogTitle>Create New Thread</DialogTitle>
                <DialogContent>
                    <Grid container spacing={2}>
                        <Grid xs={12}>
                            <Field
                                name={'title'}
                                children={(props) => {
                                    return <TextFieldSimple {...props} label={'Title'} />;
                                }}
                            />
                        </Grid>
                        <Grid xs={12}>
                            <Field
                                name={'body_md'}
                                children={(props) => {
                                    return <MarkdownField {...props} label={'Message'} />;
                                }}
                            />
                        </Grid>
                        <Grid xs={12}>
                            <Field
                                name={'sticky'}
                                children={(props) => {
                                    return <CheckboxSimple {...props} label={'Stickied'} />;
                                }}
                            />
                        </Grid>
                        <Grid xs={12}>
                            <Field
                                name={'locked'}
                                children={(props) => {
                                    return <CheckboxSimple {...props} label={'Locked'} />;
                                }}
                            />
                        </Grid>
                    </Grid>
                </DialogContent>
                <DialogActions>
                    <Grid container>
                        <Grid xs={12} mdOffset="auto">
                            <Subscribe
                                selector={(state) => [state.canSubmit, state.isSubmitting]}
                                children={([canSubmit, isSubmitting]) => {
                                    return (
                                        <Buttons
                                            reset={reset}
                                            canSubmit={canSubmit}
                                            isSubmitting={isSubmitting}
                                            clearLabel={'Delete Thread'}
                                            onClose={async () => {
                                                await modal.hide();
                                            }}
                                        />
                                    );
                                }}
                            />
                        </Grid>
                    </Grid>
                </DialogActions>
            </form>
        </Dialog>
    );
});
