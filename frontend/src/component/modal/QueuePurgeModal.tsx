import NiceModal, { muiDialogV5, useModal } from '@ebay/nice-modal-react';
import FilterAltIcon from '@mui/icons-material/FilterAlt';
import { Dialog, DialogActions, DialogContent, DialogTitle } from '@mui/material';
import ButtonGroup from '@mui/material/ButtonGroup';
import Grid from '@mui/material/Grid';
import Stack from '@mui/material/Stack';
import Typography from '@mui/material/Typography';
import { useMutation } from '@tanstack/react-query';
import { z } from 'zod';
import { apiQueueMessagesDelete } from '../../api';
import { useAppForm } from '../../contexts/formContext.tsx';
import { useUserFlashCtx } from '../../hooks/useUserFlashCtx.ts';
import { ChatLog } from '../../schema/playerqueue.ts';
import { Heading } from '../Heading';

export const QueuePurgeModal = NiceModal.create(({ message }: { message: ChatLog }) => {
    const modal = useModal();
    const { sendFlash, sendError } = useUserFlashCtx();

    const purge = useMutation({
        mutationKey: ['playerqueue_message', { message_id: message.message_id }],

        mutationFn: async (values: { count: number }) => {
            return await apiQueueMessagesDelete(message.message_id, values.count);
        },
        onSuccess: async (_, variables) => {
            sendFlash('success', `Purged ${variables.count} message(s) successfully`);
            modal.resolve();
            await modal.hide();
        },
        onError: sendError
    });

    const form = useAppForm({
        onSubmit: async ({ value }) => {
            purge.mutate({ count: value.count });
        },
        defaultValues: {
            count: 1
        }
    });

    return (
        <Dialog {...muiDialogV5(modal)} fullWidth maxWidth={'md'}>
            <form
                onSubmit={async (e) => {
                    e.preventDefault();
                    e.stopPropagation();
                    await form.handleSubmit();
                }}
            >
                <DialogTitle component={Heading} iconLeft={<FilterAltIcon />}>
                    Delete/Purge User Messages
                </DialogTitle>
                <DialogContent>
                    <Stack spacing={2}>
                        <Typography>
                            To delete a single message, use a count of 1, otherwise you can purge more messages if you
                            want. When purging more than one messages, only messages older than the selected message are
                            eligible for deletion. This will only delete the messages of the user who created the
                            selected message.
                        </Typography>
                        <Grid container spacing={2}>
                            <Grid size={{ xs: 8 }}>
                                <form.AppField
                                    name={'count'}
                                    validators={{
                                        onChange: z.number().min(1).max(10000)
                                    }}
                                    children={(field) => {
                                        return <field.TextField label={'How many messages to delete / purge.'} />;
                                    }}
                                />
                            </Grid>
                        </Grid>
                    </Stack>
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
