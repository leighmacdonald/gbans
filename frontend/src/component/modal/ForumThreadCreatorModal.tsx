import { useCallback } from 'react';
import NiceModal, { muiDialogV5, useModal } from '@ebay/nice-modal-react';
import { Dialog, DialogActions, DialogContent, DialogTitle } from '@mui/material';
import ButtonGroup from '@mui/material/ButtonGroup';
import Grid from '@mui/material/Grid';
import { useTheme } from '@mui/material/styles';
import useMediaQuery from '@mui/material/useMediaQuery';
import { useMutation } from '@tanstack/react-query';
import { z } from 'zod';
import { apiCreateThread, Forum, ForumThread } from '../../api/forum.ts';
import { useAppForm } from '../../contexts/formContext.tsx';
import { useUserFlashCtx } from '../../hooks/useUserFlashCtx.ts';
import { logErr } from '../../util/errors';
import { mdEditorRef } from '../form/field/MarkdownField.tsx';
import { ModalConfirm, ModalForumThreadCreator } from './index';

type ForumThreadEditorValues = {
    title: string;
    body_md: string;
    sticky: boolean;
    locked: boolean;
};

const schema = z.object({
    title: z.string().min(2),
    body_md: z.string().min(2),
    sticky: z.boolean(),
    locked: z.boolean()
});

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

    const form = useAppForm({
        onSubmit: async ({ value }) => {
            mutation.mutate({ ...value });
        },
        defaultValues: {
            title: '',
            body_md: '',
            sticky: false,
            locked: false
        },
        validators: {
            onSubmit: schema
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
                    await form.handleSubmit();
                }}
            >
                <DialogTitle>Create New Thread</DialogTitle>
                <DialogContent>
                    <Grid container spacing={2}>
                        <Grid size={{ xs: 12 }}>
                            <form.AppField
                                name={'title'}
                                children={(field) => {
                                    return <field.TextField label={'Title'} />;
                                }}
                            />
                        </Grid>
                        <Grid size={{ xs: 12 }}>
                            <form.AppField
                                name={'body_md'}
                                children={(field) => {
                                    return <field.MarkdownField label={'Message'} />;
                                }}
                            />
                        </Grid>
                        <Grid size={{ xs: 12 }}>
                            <form.AppField
                                name={'sticky'}
                                children={(field) => {
                                    return <field.CheckboxField label={'Stickied'} />;
                                }}
                            />
                        </Grid>
                        <Grid size={{ xs: 12 }}>
                            <form.AppField
                                name={'locked'}
                                children={(field) => {
                                    return <field.CheckboxField label={'Locked'} />;
                                }}
                            />
                        </Grid>
                    </Grid>
                </DialogContent>
                <DialogActions>
                    <Grid container>
                        <Grid size={{ xs: 12 }}>
                            <form.AppForm>
                                <ButtonGroup>
                                    <form.ResetButton />
                                    <form.SubmitButton />
                                </ButtonGroup>
                            </form.AppForm>
                        </Grid>
                    </Grid>
                </DialogActions>
            </form>
        </Dialog>
    );
});
