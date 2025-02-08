import NiceModal, { muiDialogV5, useModal } from '@ebay/nice-modal-react';
import FilterAltIcon from '@mui/icons-material/FilterAlt';
import { Dialog, DialogActions, DialogContent, DialogTitle } from '@mui/material';
import MenuItem from '@mui/material/MenuItem';
import Grid from '@mui/material/Unstable_Grid2';
import { useForm } from '@tanstack/react-form';
import { useMutation } from '@tanstack/react-query';
import { zodValidator } from '@tanstack/zod-form-adapter';
import { z } from 'zod';
import { apiQueueSetUserStatus, ChatStatus } from '../../api';
import { useQueueCtx } from '../../hooks/useQueueCtx.ts';
import { Heading } from '../Heading';
import { Buttons } from '../field/Buttons.tsx';
import { SelectFieldSimple } from '../field/SelectFieldSimple.tsx';
import { TextFieldSimple } from '../field/TextFieldSimple.tsx';

export const QueueStatusModal = NiceModal.create(({ steam_id }: { steam_id: string }) => {
    const modal = useModal();
    const { chatStatus, reason } = useQueueCtx();

    const mutation = useMutation({
        mutationKey: ['playerqueue_status', { steam_id }],
        mutationFn: async (values: { chat_status: ChatStatus; reason: string }) => {
            return await apiQueueSetUserStatus(steam_id, values.chat_status, values.reason);
        },
        onSuccess: async (result) => {
            modal.resolve(result);
            await modal.hide();
        }
    });

    const { Field, Subscribe, handleSubmit, reset } = useForm({
        onSubmit: async ({ value }) => {
            mutation.mutate(value);
        },
        validatorAdapter: zodValidator,
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
                        <Grid xs={2}>
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

                        <Grid xs={10}>
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
