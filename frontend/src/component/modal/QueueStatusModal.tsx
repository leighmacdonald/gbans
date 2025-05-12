import NiceModal, { muiDialogV5, useModal } from '@ebay/nice-modal-react';
import FilterAltIcon from '@mui/icons-material/FilterAlt';
import { Dialog, DialogActions, DialogContent, DialogTitle } from '@mui/material';
import ButtonGroup from '@mui/material/ButtonGroup';
import Grid from '@mui/material/Grid';
import MenuItem from '@mui/material/MenuItem';
import { useMutation } from '@tanstack/react-query';
import { z } from 'zod';
import { apiQueueSetUserStatus, ChatStatus } from '../../api';
import { useAppForm } from '../../contexts/formContext.tsx';
import { useQueueCtx } from '../../hooks/useQueueCtx.ts';
import { useUserFlashCtx } from '../../hooks/useUserFlashCtx.ts';
import { Heading } from '../Heading';

const schema = z.object({
    chat_status: z.enum(['readwrite', 'readonly', 'noaccess']),
    reason: z.string({ message: 'Reason' })
});

export const QueueStatusModal = NiceModal.create(({ steam_id }: { steam_id: string }) => {
    const modal = useModal();
    const { chatStatus, reason } = useQueueCtx();
    const { sendError } = useUserFlashCtx();

    const mutation = useMutation({
        mutationKey: ['playerqueue_status', { steam_id }],
        mutationFn: async (values: { chat_status: ChatStatus; reason: string }) => {
            return await apiQueueSetUserStatus(steam_id, values.chat_status, values.reason);
        },
        onSuccess: async (result) => {
            modal.resolve(result);
            await modal.hide();
        },
        onError: sendError
    });

    const form = useAppForm({
        onSubmit: async ({ value }) => {
            mutation.mutate(value);
        },
        defaultValues: {
            chat_status: chatStatus,
            reason: reason
        },
        validators: {
            onSubmit: schema
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
                    <Grid container spacing={2}>
                        <Grid size={{ xs: 2 }}>
                            <form.AppField
                                name={'chat_status'}
                                children={(field) => {
                                    return (
                                        <field.SelectField
                                            label={'Chat Status'}
                                            items={['readwrite', 'readonly', 'noaccess']}
                                            renderItem={(du) => {
                                                return (
                                                    <MenuItem value={du} key={`du-${du}`}>
                                                        {du}
                                                    </MenuItem>
                                                );
                                            }}
                                        />
                                    );
                                }}
                            />
                        </Grid>

                        <Grid size={{ xs: 10 }}>
                            <form.AppField
                                name={'reason'}
                                children={(field) => {
                                    return <field.TextField label={'Reason for status change'} />;
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
