import NiceModal, { muiDialogV5, useModal } from '@ebay/nice-modal-react';
import CloudDoneIcon from '@mui/icons-material/CloudDone';
import { Dialog, DialogActions, DialogContent, DialogTitle } from '@mui/material';
import ButtonGroup from '@mui/material/ButtonGroup';
import Grid from '@mui/material/Grid';
import { useMutation } from '@tanstack/react-query';
import { z } from 'zod/v4';
import { apiCreateWhitelistIP, apiUpdateWhitelistIP } from '../../api';
import { useAppForm } from '../../contexts/formContext.tsx';
import { useUserFlashCtx } from '../../hooks/useUserFlashCtx.ts';
import { WhitelistIP } from '../../schema/network.ts';
import { Heading } from '../Heading';

export const IPWhitelistEditorModal = NiceModal.create(({ source }: { source?: WhitelistIP }) => {
    const modal = useModal();
    const { sendError } = useUserFlashCtx();

    const mutation = useMutation({
        mutationKey: ['blockSource'],
        mutationFn: async (values: { address: string }) => {
            if (source?.cidr_block_whitelist_id) {
                const resp = await apiUpdateWhitelistIP(source.cidr_block_whitelist_id, values.address);
                modal.resolve(resp);
            } else {
                const resp = await apiCreateWhitelistIP(values.address);
                modal.resolve(resp);
            }
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
        defaultValues: {
            address: source?.address ?? ''
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
                    CIDR Block Whitelist Editor
                </DialogTitle>
                <DialogContent>
                    <Grid container spacing={2}>
                        <Grid size={{ xs: 12 }}>
                            <form.AppField
                                name={'address'}
                                validators={{
                                    onChange: z.string().refine((arg) => {
                                        const pieces = arg.split('/');
                                        const addr = pieces[0];
                                        const result = z.ipv4().safeParse(addr);
                                        return result.success;
                                    })
                                }}
                                children={(field) => {
                                    return <field.TextField label={'IP Addr'} />;
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
