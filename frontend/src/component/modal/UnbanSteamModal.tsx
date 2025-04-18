import NiceModal, { muiDialogV5, useModal } from '@ebay/nice-modal-react';
import { Dialog, DialogActions, DialogContent, DialogTitle } from '@mui/material';
import Grid from '@mui/material/Unstable_Grid2';
import { useForm } from '@tanstack/react-form';
import { useMutation } from '@tanstack/react-query';
import { z } from 'zod';
import { apiDeleteBan } from '../../api';
import { useUserFlashCtx } from '../../hooks/useUserFlashCtx.ts';
import { Buttons } from '../field/Buttons.tsx';
import { TextFieldSimple } from '../field/TextFieldSimple.tsx';

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

        const { Field, Subscribe, handleSubmit, reset } = useForm({
            onSubmit: async ({ value }) => {
                mutation.mutate(value.unban_reason);
            },
            validators: {
                onChange: z.object({
                    unban_reason: z.string().min(4, 'Min length 4')
                })
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
                    <DialogTitle>
                        Unban {personaName} (#{banId})
                    </DialogTitle>

                    <DialogContent>
                        <Grid container spacing={2}>
                            <Grid xs={12}>
                                <Field
                                    name={'unban_reason'}
                                    children={(props) => {
                                        return (
                                            <TextFieldSimple
                                                {...props}
                                                error={props.state.meta.errors.length > 0}
                                                errorText={props.state.meta.errors
                                                    .map((e) => (e ? e.message : null))
                                                    .filter((f) => f)
                                                    .join(', ')}
                                                label={'Unban Reason'}
                                            />
                                        );
                                    }}
                                />
                            </Grid>
                        </Grid>
                    </DialogContent>

                    <DialogActions>
                        <Grid container>
                            <Grid xs={12}>
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
