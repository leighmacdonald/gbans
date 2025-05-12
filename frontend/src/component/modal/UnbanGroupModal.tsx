import NiceModal, { muiDialogV5, useModal } from '@ebay/nice-modal-react';
import { Dialog, DialogActions, DialogContent, DialogTitle } from '@mui/material';
import ButtonGroup from '@mui/material/ButtonGroup';
import Grid from '@mui/material/Grid';
import { useMutation } from '@tanstack/react-query';
import { z } from 'zod';
import { apiDeleteGroupBan } from '../../api';
import { useAppForm } from '../../contexts/formContext.tsx';
import { useUserFlashCtx } from '../../hooks/useUserFlashCtx.ts';

export const UnbanGroupModal = NiceModal.create(
    ({
        banId
    }: {
        banId: number; // common placeholder for any primary key id for a ban
    }) => {
        const modal = useModal();
        const { sendError } = useUserFlashCtx();

        const mutation = useMutation({
            mutationKey: ['deleteGroupBan', { banId }],
            mutationFn: async (unban_reason: string) => {
                await apiDeleteGroupBan(banId, unban_reason);
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
            defaultValues: {
                unban_reason: ''
            },
            validators: {
                onSubmit: z.object({
                    unban_reason: z.string().min(5)
                })
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
                    <DialogTitle>Unban Steam Group (#{banId})</DialogTitle>

                    <DialogContent>
                        <Grid container>
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
                                        <form.CloseButton
                                            onClick={async () => {
                                                await modal.hide();
                                            }}
                                        />
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
