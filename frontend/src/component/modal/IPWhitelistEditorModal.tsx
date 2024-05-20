import NiceModal, { muiDialogV5, useModal } from '@ebay/nice-modal-react';
import CloudDoneIcon from '@mui/icons-material/CloudDone';
import { Dialog, DialogActions, DialogContent, DialogTitle } from '@mui/material';
import Grid from '@mui/material/Unstable_Grid2';
import { useForm } from '@tanstack/react-form';
import { useMutation } from '@tanstack/react-query';
import { zodValidator } from '@tanstack/zod-form-adapter';
import { z } from 'zod';
import { apiCreateWhitelistIP, apiUpdateWhitelistIP, WhitelistIP } from '../../api';
import { Heading } from '../Heading';
import { Buttons } from '../field/Buttons.tsx';
import { TextFieldSimple } from '../field/TextFieldSimple.tsx';

export const IPWhitelistEditorModal = NiceModal.create(({ source }: { source?: WhitelistIP }) => {
    const modal = useModal();

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
            modal.reject(error);
        }
    });

    const { Field, Subscribe, handleSubmit, reset } = useForm({
        onSubmit: async ({ value }) => {
            mutation.mutate(value);
        },
        validatorAdapter: zodValidator,
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
                    await handleSubmit();
                }}
            >
                <DialogTitle component={Heading} iconLeft={<CloudDoneIcon />}>
                    CIDR Block Whitelist Editor
                </DialogTitle>
                <DialogContent>
                    <Grid container spacing={2}>
                        <Grid xs={12}>
                            <Field
                                name={'address'}
                                validators={{
                                    onChange: z.string().refine((arg) => {
                                        const pieces = arg.split('/');
                                        const addr = pieces[0];
                                        const result = z.string().ip(addr).safeParse(addr);
                                        return result.success;
                                    })
                                }}
                                children={(props) => {
                                    return <TextFieldSimple {...props} label={'IP Addr'} />;
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
});
