import { useCallback } from 'react';
import NiceModal, { muiDialogV5, useModal } from '@ebay/nice-modal-react';
import DeleteForeverIcon from '@mui/icons-material/DeleteForever';
import Button from '@mui/material/Button';
import Dialog from '@mui/material/Dialog';
import DialogActions from '@mui/material/DialogActions';
import DialogContent from '@mui/material/DialogContent';
import DialogTitle from '@mui/material/DialogTitle';
import Stack from '@mui/material/Stack';
import { apiDeleteThread, ForumThread } from '../../api/forum';
import { useUserFlashCtx } from '../../hooks/useUserFlashCtx.ts';
import { logErr } from '../../util/errors';
import { ModalConfirm } from './index';

// interface ThreadEditValues {
//     title: string;
//     sticky: boolean;
//     locked: boolean;
// }

// const validationSchema = yup.object({
//     title: titleFieldValidator,
//     sticky: yup.boolean().required(),
//     locked: yup.boolean().required()
// });

export const ForumThreadEditorModal = NiceModal.create(({ thread }: { thread: ForumThread }) => {
    const modal = useModal();
    const confirmModal = useModal(ModalConfirm);
    const { sendFlash } = useUserFlashCtx();

    // const onSubmit = useCallback(
    //     async (values: ThreadEditValues) => {
    //         const abortController = new AbortController();
    //         try {
    //             const newThread = await apiUpdateThread(
    //                 thread.forum_thread_id,
    //                 values.title,
    //                 values.sticky,
    //                 values.locked,
    //                 abortController
    //             );
    //             modal.resolve(newThread);
    //         } catch (e) {
    //             modal.reject(e);
    //         } finally {
    //             await modal.hide();
    //         }
    //     },
    //     [modal, thread.forum_thread_id]
    // );

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

    return (
        // <Formik<ThreadEditValues>
        //     initialValues={{
        //         title: thread.title,
        //         locked: thread.locked,
        //         sticky: thread.sticky
        //     }}
        //     onSubmit={onSubmit}
        //     validationSchema={validationSchema}
        // >
        <Dialog {...muiDialogV5(modal)} fullWidth>
            <DialogTitle>{`Edit Thread #${thread.forum_thread_id}`}</DialogTitle>

            <DialogContent>
                <Stack spacing={2}>
                    {/*<TitleField />*/}
                    {/*<StickyField />*/}
                    {/*<LockedField />*/}
                </Stack>
            </DialogContent>

            <DialogActions>
                <Button startIcon={<DeleteForeverIcon />} color={'error'} variant={'contained'} onClick={onDelete}>
                    Delete
                </Button>
                {/*<CancelButton />*/}
                {/*<SubmitButton />*/}
            </DialogActions>
        </Dialog>
        // </Formik>
    );
});

export default ForumThreadEditorModal;
