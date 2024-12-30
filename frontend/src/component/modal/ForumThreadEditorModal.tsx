import { useCallback } from 'react';
import NiceModal, { muiDialogV5, useModal } from '@ebay/nice-modal-react';
import Dialog from '@mui/material/Dialog';
import DialogActions from '@mui/material/DialogActions';
import DialogContent from '@mui/material/DialogContent';
import DialogTitle from '@mui/material/DialogTitle';
import Grid from '@mui/material/Grid2';
import { useForm } from '@tanstack/react-form';
import { useMutation } from '@tanstack/react-query';
import { apiDeleteThread, apiUpdateThread, ForumThread } from '../../api/forum';
import { useUserFlashCtx } from '../../hooks/useUserFlashCtx.ts';
import { logErr } from '../../util/errors';
import { Buttons } from '../field/Buttons.tsx';
import { CheckboxSimple } from '../field/CheckboxSimple.tsx';
import { TextFieldSimple } from '../field/TextFieldSimple.tsx';
import { ModalConfirm } from './index';

type ThreadEditValues = {
    title: string;
    sticky: boolean;
    locked: boolean;
};

export const ForumThreadEditorModal = NiceModal.create(({ thread }: { thread: ForumThread }) => {
    const modal = useModal();
    const confirmModal = useModal(ModalConfirm);
    const { sendFlash, sendError } = useUserFlashCtx();

    const onDelete = useCallback(async () => {
        const abortController = new AbortController();
        try {
            const confirmed = await confirmModal.show({
                title: 'Confirm Thread Deletion',
                children: 'All messages will be deleted'
            });
            if (confirmed) {
                await confirmModal.hide();
                await apiDeleteThread(thread.forum_thread_id, abortController);
                thread.forum_thread_id = 0;
                modal.resolve(thread);
                await modal.hide();
                sendFlash('success', 'Deleted thread successfully');
            } else {
                await confirmModal.hide();
            }
        } catch (e) {
            logErr(e);
        }
    }, [confirmModal, modal, sendFlash, thread]);

    const mutation = useMutation({
        mutationKey: ['forumThread', { forum_thread_id: thread.forum_thread_id }],
        mutationFn: async (values: ThreadEditValues) => {
            return await apiUpdateThread(thread.forum_thread_id, values.title, values.sticky, values.locked);
        },
        onSuccess: async (editedThread: ForumThread) => {
            modal.resolve(editedThread);
            await modal.hide();
        },
        onError: sendError
    });

    const { Field, Subscribe, handleSubmit, reset } = useForm({
        onSubmit: async ({ value }) => {
            mutation.mutate({ ...value });
        },
        defaultValues: {
            title: thread.title,
            sticky: thread.sticky,
            locked: thread.locked
        }
    });

    return (
        <Dialog {...muiDialogV5(modal)} fullWidth>
            <form
                onSubmit={async (e) => {
                    e.preventDefault();
                    e.stopPropagation();
                    await handleSubmit();
                }}
            >
                <DialogTitle>{`Edit Thread #${thread.forum_thread_id}`}</DialogTitle>

                <DialogContent>
                    <Grid container spacing={2}>
                        <Grid size={{ xs: 12 }}>
                            <Field
                                name={'title'}
                                children={(props) => {
                                    return <TextFieldSimple {...props} label={'Title'} />;
                                }}
                            />
                        </Grid>
                        <Grid size={{ xs: 12 }}>
                            <Field
                                name={'sticky'}
                                children={(props) => {
                                    return <CheckboxSimple {...props} label={'Stickied'} />;
                                }}
                            />
                        </Grid>
                        <Grid size={{ xs: 12 }}>
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
                        <Grid size={{ xs: 12 }}>
                            <Subscribe
                                selector={(state) => [state.canSubmit, state.isSubmitting]}
                                children={([canSubmit, isSubmitting]) => {
                                    return (
                                        <Buttons
                                            reset={reset}
                                            canSubmit={canSubmit}
                                            isSubmitting={isSubmitting}
                                            clearLabel={'Delete Thread'}
                                            onClear={onDelete}
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
