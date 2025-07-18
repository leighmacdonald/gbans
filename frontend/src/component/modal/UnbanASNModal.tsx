import NiceModal, { muiDialogV5, useModal } from '@ebay/nice-modal-react';
import { Dialog, DialogActions, DialogContent, DialogTitle } from '@mui/material';
import ButtonGroup from '@mui/material/ButtonGroup';
import Grid from '@mui/material/Grid';
import { useMutation } from '@tanstack/react-query';
import { z } from 'zod/v4';
import { apiDeleteASNBan } from '../../api';
import { useAppForm } from '../../contexts/formContext.tsx';
import { useUserFlashCtx } from '../../hooks/useUserFlashCtx.ts';

const schema = z.object({
    unban_reason: z.string().min(5)
});

export const UnbanASNModal = NiceModal.create(({ banId }: { banId: number }) => {
    const modal = useModal();
    const { sendError } = useUserFlashCtx();
    const defaultValues: z.input<typeof schema> = {
        unban_reason: ''
    };
    const mutation = useMutation({
        mutationKey: ['deleteASNBan', { banId }],
        mutationFn: async (unban_reason: string) => {
            await apiDeleteASNBan(banId, unban_reason);
        },
        onSuccess: async () => {
            modal.resolve();
            await modal.hide();
        },
        onError: (error) => {
            sendError(error);
            modal.reject(error);
        }
    });

    const form = useAppForm({
        onSubmit: async ({ value }) => {
            mutation.mutate(value.unban_reason);
        },
        defaultValues,
        validators: {
            onSubmit: schema
        }
    });

    return (
        <Dialog {...muiDialogV5(modal)}>
            <form
                onSubmit={async (e) => {
                    e.preventDefault();
                    e.stopPropagation();
                    await form.handleSubmit();
                }}
            >
                <DialogTitle>Unban ASN (#{banId})</DialogTitle>

                <DialogContent>
                    <Grid container spacing={2}>
                        <Grid size={{ xs: 12 }}>
                            <form.AppField
                                name={'unban_reason'}
                                validators={{
                                    onChange: z.string().min(5)
                                }}
                                children={(field) => {
                                    return <field.TextField label={'Unban Reason'} />;
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
                                    <form.CloseButton />
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
