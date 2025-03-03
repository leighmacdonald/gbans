import NiceModal, { muiDialogV5, useModal } from '@ebay/nice-modal-react';
import BlockIcon from '@mui/icons-material/Block';
import { Dialog, DialogActions, DialogContent, DialogTitle } from '@mui/material';
import Grid from '@mui/material/Unstable_Grid2';
import { useForm } from '@tanstack/react-form';
import { useMutation } from '@tanstack/react-query';
import { z } from 'zod';
import { apiCreateCIDRBlockSource, apiUpdateCIDRBlockSource, CIDRBlockSource } from '../../api';
import { useUserFlashCtx } from '../../hooks/useUserFlashCtx.ts';
import { Heading } from '../Heading';
import { Buttons } from '../field/Buttons.tsx';
import { CheckboxSimple } from '../field/CheckboxSimple.tsx';
import { TextFieldSimple } from '../field/TextFieldSimple.tsx';

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

    const { Field, Subscribe, handleSubmit, reset } = useForm({
        onSubmit: async ({ value }) => {
            mutation.mutate(value);
        },
        validators: {
            onChange: z.object({
                cidr_block_source_id: z.number(),
                name: z.string().min(2),
                url: z.string().url(),
                enabled: z.boolean()
            })
        },
        defaultValues: {
            cidr_block_source_id: source?.cidr_block_source_id ?? 0,
            name: source?.name ?? '',
            url: source?.url ?? '',
            enabled: source?.enabled ?? true
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
                <DialogTitle component={Heading} iconLeft={<BlockIcon />}>
                    CIDR Block Source Editor
                </DialogTitle>
                <DialogContent>
                    <Grid container spacing={2}>
                        <Grid xs={12}>
                            <Field
                                name={'name'}
                                children={(props) => {
                                    return <TextFieldSimple {...props} label={'Source Name'} />;
                                }}
                            />
                            <Grid xs={12}>
                                <Field
                                    name={'url'}
                                    children={(props) => {
                                        return <TextFieldSimple {...props} label={'Source URL'} />;
                                    }}
                                />
                            </Grid>
                            <Grid xs={12}>
                                <Field
                                    name={'enabled'}
                                    children={({ state, handleBlur, handleChange }) => {
                                        return (
                                            <CheckboxSimple
                                                checked={state.value}
                                                onChange={(_, v) => handleChange(v)}
                                                onBlur={handleBlur}
                                                label={'Enabled'}
                                            />
                                        );
                                    }}
                                />
                            </Grid>
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
});
