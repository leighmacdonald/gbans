import NiceModal, { muiDialogV5, useModal } from '@ebay/nice-modal-react';
import BlockIcon from '@mui/icons-material/Block';
import { Dialog, DialogActions, DialogContent, DialogTitle } from '@mui/material';
import Grid from '@mui/material/Grid';
import { useMutation } from '@tanstack/react-query';
import { z } from 'zod';
import { apiCreateCIDRBlockSource, apiUpdateCIDRBlockSource, CIDRBlockSource } from '../../api';
import { useAppForm } from '../../contexts/formContext.tsx';
import { useUserFlashCtx } from '../../hooks/useUserFlashCtx.ts';
import { Heading } from '../Heading';

interface CIDRBlockEditorValues {
    name: string;
    url: string;
    enabled: boolean;
}

export const CIDRBlockEditorModal = NiceModal.create(({ source }: { source?: CIDRBlockSource }) => {
    const modal = useModal();
    const { sendError } = useUserFlashCtx();

    const mutation = useMutation({
        mutationKey: ['blockSource'],
        mutationFn: async (values: CIDRBlockEditorValues) => {
            if (source?.cidr_block_source_id) {
                const resp = await apiUpdateCIDRBlockSource(
                    source.cidr_block_source_id,
                    values.name,
                    values.url,
                    values.enabled
                );
                modal.resolve(resp);
            } else {
                const resp = await apiCreateCIDRBlockSource(values.name, values.url, values.enabled);
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
            name: source?.name ?? '',
            url: source?.url ?? '',
            enabled: source?.enabled ?? true
        },
        validators: {
            onSubmit: z.object({
                name: z.string().min(2),
                url: z.string().url(),
                enabled: z.boolean()
            })
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
                <DialogTitle component={Heading} iconLeft={<BlockIcon />}>
                    CIDR Block Source Editor
                </DialogTitle>
                <DialogContent>
                    <Grid container spacing={2}>
                        <Grid size={{ xs: 12 }}>
                            <form.AppField
                                name={'name'}
                                children={(field) => {
                                    return <field.TextField label={'Source Name'} />;
                                }}
                            />
                        </Grid>
                        <Grid size={{ xs: 12 }}>
                            <form.AppField
                                name={'url'}
                                children={(field) => {
                                    return <field.TextField label={'Source URL'} />;
                                }}
                            />
                        </Grid>
                        <Grid size={{ xs: 12 }}>
                            <form.AppField
                                name={'enabled'}
                                children={(field) => {
                                    return <field.CheckboxField label={'Enabled'} />;
                                }}
                            />
                        </Grid>
                    </Grid>
                </DialogContent>
                <DialogActions>
                    <Grid container>
                        <Grid size={{ xs: 12 }}>
                            <form.AppForm>
                                <form.ResetButton />
                                <form.SubmitButton />
                            </form.AppForm>
                        </Grid>
                    </Grid>
                </DialogActions>
            </form>
        </Dialog>
    );
});
