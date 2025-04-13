import NiceModal, { muiDialogV5, useModal } from '@ebay/nice-modal-react';
import { Dialog, DialogActions, DialogContent, DialogTitle } from '@mui/material';
import Grid from '@mui/material/Grid';
import { useForm } from '@tanstack/react-form';
import { useMutation } from '@tanstack/react-query';
import { z } from 'zod';
import { apiDeleteCIDRBan } from '../../api';
import { useUserFlashCtx } from '../../hooks/useUserFlashCtx.ts';
import { Buttons } from '../field/Buttons.tsx';
import { TextFieldSimple } from '../field/TextFieldSimple.tsx';

export const UnbanCIDRModal = NiceModal.create(
    ({
        banId
    }: {
        banId: number; // common placeholder for any primary key id for a ban
        personaName?: string;
    }) => {
        const modal = useModal();
        const { sendError } = useUserFlashCtx();

        const mutation = useMutation({
            mutationKey: ['deleteCIDRBan', { banId }],
            mutationFn: async (unban_reason: string) => {
                await apiDeleteCIDRBan(banId, unban_reason);
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

        const { Field, Subscribe, handleSubmit, reset } = useForm({
            onSubmit: async ({ value }) => {
                mutation.mutate(value.unban_reason);
            },
            defaultValues: {
                unban_reason: ''
            }
        });

        return (
            <Dialog {...muiDialogV5(modal)}>
                <form
                    onSubmit={async (e) => {
                        e.preventDefault();
                        e.stopPropagation();
                        await handleSubmit();
                    }}
                >
                    <DialogTitle>Unban CIDR (#{banId})</DialogTitle>

                    <DialogContent>
                        <Grid container>
                            <Grid size={{ xs: 12 }}>
                                <Field
                                    name={'unban_reason'}
                                    validators={{
                                        onChange: z.string().min(5)
                                    }}
                                    children={(props) => {
                                        return <TextFieldSimple {...props} label={'Unban Reason'} />;
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
                                        return (
                                            <Buttons
                                                reset={reset}
                                                canSubmit={canSubmit}
                                                isSubmitting={isSubmitting}
                                                onClose={async () => {
                                                    await modal.hide();
                                                }}
                                            />
                                        );
                                    }}
                                />
                            </Grid>
                        </Grid>
                    </DialogActions>
                </form>
            </Dialog>
        );
    }
);
