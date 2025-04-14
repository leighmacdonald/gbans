import NiceModal, { muiDialogV5, useModal } from '@ebay/nice-modal-react';
import FilterAltIcon from '@mui/icons-material/FilterAlt';
import { Dialog, DialogActions, DialogContent, DialogTitle } from '@mui/material';
import Grid from '@mui/material/Grid';
import MenuItem from '@mui/material/MenuItem';
import { useForm } from '@tanstack/react-form';
import { useMutation } from '@tanstack/react-query';
import { z } from 'zod';
import { apiQueueSetUserStatus, ChatStatus } from '../../api';
import { useQueueCtx } from '../../hooks/useQueueCtx.ts';
import { useUserFlashCtx } from '../../hooks/useUserFlashCtx.ts';
import { Heading } from '../Heading';
import { Buttons } from '../field/Buttons.tsx';
import { SelectFieldSimple } from '../field/SelectFieldSimple.tsx';
import { TextFieldSimple } from '../field/TextFieldSimple.tsx';

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

    const { Field, Subscribe, handleSubmit, reset } = useForm({
        onSubmit: async ({ value }) => {
            mutation.mutate(value);
        },
        defaultValues: {
            chat_status: chatStatus,
            reason: reason
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
                    <Grid container spacing={2}>
                        <Grid size={{ xs: 2 }}>
                            <Field
                                name={'chat_status'}
                                validators={{
                                    onChange: z.enum(['readwrite', 'readonly', 'noaccess'])
                                }}
                                children={(props) => {
                                    return (
                                        <SelectFieldSimple
                                            {...props}
                                            label={'Chat Status'}
                                            fullwidth={true}
                                            items={['readwrite', 'readonly', 'noaccess']}
                                            renderMenu={(du) => {
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
                            <Field
                                name={'reason'}
                                validators={{
                                    onChange: z.string({ message: 'Reason' })
                                }}
                                children={(props) => {
                                    return <TextFieldSimple {...props} label={'Reason for status change'} />;
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
