import NiceModal, { muiDialogV5, useModal } from '@ebay/nice-modal-react';
import FilterAltIcon from '@mui/icons-material/FilterAlt';
import { Dialog, DialogActions, DialogContent, DialogTitle } from '@mui/material';
import Stack from '@mui/material/Stack';
import Typography from '@mui/material/Typography';
import Grid from '@mui/material/Unstable_Grid2';
import { useForm } from '@tanstack/react-form';
import { useMutation } from '@tanstack/react-query';
import { zodValidator } from '@tanstack/zod-form-adapter';
import { z } from 'zod';
import { apiQueueMessagesDelete, ChatLog } from '../../api';
import { useUserFlashCtx } from '../../hooks/useUserFlashCtx.ts';
import { Heading } from '../Heading';
import { Buttons } from '../field/Buttons.tsx';
import { TextFieldSimple } from '../field/TextFieldSimple.tsx';

export const QueuePurgeModal = NiceModal.create(({ message }: { message: ChatLog }) => {
    const modal = useModal();
    const { sendFlash } = useUserFlashCtx();

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
        onError: (error) => {
            sendFlash('error', `Error trying to purge message(s): ${error}`);
        }
    });

    const { Field, Subscribe, handleSubmit, reset } = useForm({
        onSubmit: async ({ value }) => {
            purge.mutate({
                count: Number(value.count)
            });
        },
        validatorAdapter: zodValidator,
        defaultValues: {
            count: '1'
        }
    });

    return (
        <Dialog {...muiDialogV5(modal)} fullWidth maxWidth={'md'}>
            <form
                onSubmit={async (e) => {
                    e.preventDefault();
                    e.stopPropagation();
                    await handleSubmit();
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
                            eligable for deletion. This will only delete the messages of the user who created the
                            selected message.
                        </Typography>
                        <Grid container spacing={2}>
                            <Grid xs={8}>
                                <Field
                                    name={'count'}
                                    validators={{
                                        onChange: z
                                            .number({ coerce: true, message: 'Must enter positive number' })
                                            .min(1)
                                            .max(10000)
                                    }}
                                    children={(props) => {
                                        return (
                                            <TextFieldSimple
                                                {...props}
                                                label={'How many messages to delete / purge.'}
                                            />
                                        );
                                    }}
                                />
                            </Grid>
                        </Grid>
                    </Stack>
                </DialogContent>
                <DialogActions>
                    <Grid container>
                        <Grid xs={12} mdOffset="auto">
                            <Subscribe
                                selector={(state) => [state.canSubmit, state.isSubmitting]}
                                children={([canSubmit, isSubmitting]) => {
                                    return <Buttons reset={reset} canSubmit={canSubmit} isSubmitting={isSubmitting} />;
                                }}
                            />
                        </Grid>
                    </Grid>
                </DialogActions>
            </form>
        </Dialog>
    );
});
