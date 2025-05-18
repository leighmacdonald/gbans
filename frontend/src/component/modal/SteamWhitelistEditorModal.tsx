import NiceModal, { muiDialogV5, useModal } from '@ebay/nice-modal-react';
import CloudDoneIcon from '@mui/icons-material/CloudDone';
import { Dialog, DialogActions, DialogContent, DialogTitle } from '@mui/material';
import ButtonGroup from '@mui/material/ButtonGroup';
import Grid from '@mui/material/Grid';
import { useMutation } from '@tanstack/react-query';
import { z } from 'zod';
import { apiCreateWhitelistSteam } from '../../api';
import { useAppForm } from '../../contexts/formContext.tsx';
import { useUserFlashCtx } from '../../hooks/useUserFlashCtx.ts';
import { Heading } from '../Heading';

const schema = z.object({
    steam_id: z.string()
});

type ValueType = z.infer<typeof schema>;

export const SteamWhitelistEditorModal = NiceModal.create(() => {
    const modal = useModal();
    const { sendError } = useUserFlashCtx();
    const defaultValues: z.input<typeof schema> = {
        steam_id: ''
    };

    const mutation = useMutation({
        mutationKey: ['blockSourceSteam'],
        mutationFn: async ({ steam_id }: ValueType) => {
            const resp = await apiCreateWhitelistSteam(steam_id);
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

    const form = useAppForm({
        onSubmit: async ({ value }) => {
            mutation.mutate(value);
        },
        defaultValues,
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
                <DialogTitle component={Heading} iconLeft={<CloudDoneIcon />}>
                    Steam Whitelist Editor
                </DialogTitle>
                <DialogContent>
                    <Grid container spacing={2}>
                        <Grid size={{ xs: 12 }}>
                            <form.AppField
                                name={'steam_id'}
                                children={(field) => {
                                    return <field.SteamIDField label={'Steam ID'} />;
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
