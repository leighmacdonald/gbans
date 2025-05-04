import NiceModal, { muiDialogV5, useModal } from '@ebay/nice-modal-react';
import CloudDoneIcon from '@mui/icons-material/CloudDone';
import { Dialog, DialogActions, DialogContent, DialogTitle } from '@mui/material';
import Grid from '@mui/material/Grid';
import { useForm } from '@tanstack/react-form';
import { useMutation } from '@tanstack/react-query';
import { apiCreateWhitelistSteam } from '../../api';
import { useUserFlashCtx } from '../../hooks/useUserFlashCtx.ts';
import { Heading } from '../Heading';
import { Buttons } from '../field/Buttons.tsx';
import { SteamIDField } from '../field/SteamIDField.tsx';

export const SteamWhitelistEditorModal = NiceModal.create(() => {
    const modal = useModal();
    const { sendError } = useUserFlashCtx();

    const mutation = useMutation({
        mutationKey: ['blockSourceSteam'],
        mutationFn: async (values: { steam_id: string }) => {
            const resp = await apiCreateWhitelistSteam(values.steam_id);
            modal.resolve(resp);
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
            mutation.mutate(value);
        },
        defaultValues: {
            steam_id: ''
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
                <DialogTitle component={Heading} iconLeft={<CloudDoneIcon />}>
                    Steam Whitelist Editor
                </DialogTitle>
                <DialogContent>
                    <Grid container spacing={2}>
                        <Grid size={{ xs: 12 }}>
                            <Field
                                name={'steam_id'}
                                // validators={makeSteamidValidators()}
                                children={(props) => {
                                    return (
                                        <SteamIDField
                                            {...props}
                                            value={props.state.value}
                                            label={'Steam ID'}
                                            fullwidth
                                        />
                                    );
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
});
