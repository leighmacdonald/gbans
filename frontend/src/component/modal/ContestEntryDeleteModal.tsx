import { useCallback } from 'react';
import NiceModal, { muiDialogV5, useModal } from '@ebay/nice-modal-react';
import {
    Dialog,
    DialogActions,
    DialogContent,
    DialogTitle
} from '@mui/material';
import Stack from '@mui/material/Stack';
import Typography from '@mui/material/Typography';
import { Formik } from 'formik';
import { apiContestEntryDelete } from '../../api';
import { CancelButton, SubmitButton } from './Buttons';

export const ContestEntryDeleteModal = NiceModal.create(
    ({ contest_entry_id }: { contest_entry_id: string }) => {
        const modal = useModal();

        const onSubmit = useCallback(async () => {
            try {
                await apiContestEntryDelete(contest_entry_id);
                modal.resolve();
            } catch (e) {
                modal.reject(e);
            } finally {
                await modal.hide();
            }
        }, [contest_entry_id, modal]);

        return (
            <Formik initialValues={{}} onSubmit={onSubmit}>
                <Dialog {...muiDialogV5(modal)}>
                    <DialogTitle>
                        Are you sure you want to delete contest entry? (
                        {contest_entry_id})
                    </DialogTitle>

                    <DialogContent>
                        <Stack spacing={2}>
                            <Typography variant={'body1'}>
                                This is irreversible and will also remove user
                                vote history for the entry
                            </Typography>
                        </Stack>
                    </DialogContent>

                    <DialogActions>
                        <CancelButton />
                        <SubmitButton />
                    </DialogActions>
                </Dialog>
            </Formik>
        );
    }
);

export default ContestEntryDeleteModal;
