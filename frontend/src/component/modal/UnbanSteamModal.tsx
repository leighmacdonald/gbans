import NiceModal, { muiDialogV5, useModal } from '@ebay/nice-modal-react';
import { Dialog, DialogActions, DialogContent, DialogTitle } from '@mui/material';
import ButtonGroup from '@mui/material/ButtonGroup';
import Grid from '@mui/material/Grid';
import { useMutation } from '@tanstack/react-query';
import { z } from 'zod/v4';
import { apiDeleteBan } from '../../api';
import { useAppForm } from '../../contexts/formContext.tsx';
import { useUserFlashCtx } from '../../hooks/useUserFlashCtx.ts';

const onSubmit = z.object({
    unban_reason: z.string().min(5)
});

export const UnbanSteamModal = NiceModal.create(
    ({
        banId,
        personaName
    }: {
        banId: number; // common placeholder for any primary key id for a ban
        personaName?: string;
    }) => {
        const modal = useModal();
        const { sendError } = useUserFlashCtx();

        const defaultValues: z.input<typeof onSubmit> = {
            unban_reason: ''
        };

        const mutation = useMutation({
            mutationKey: ['deleteSteamBan', { banId }],
            mutationFn: async (unban_reason: string) => {
                await apiDeleteBan(banId, unban_reason);
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
            validators: { onSubmit }
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
                    <DialogTitle>
                        Unban {personaName} (#{banId})
                    </DialogTitle>

                    <DialogContent>
                        <Grid container spacing={2}>
                            <Grid size={{ xs: 12 }}>
                                <form.AppField
                                    name={'unban_reason'}
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
    }
);
