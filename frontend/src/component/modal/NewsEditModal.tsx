import NiceModal, { muiDialogV5, useModal } from '@ebay/nice-modal-react';
import PersonIcon from '@mui/icons-material/Person';
import { Dialog, DialogActions, DialogContent, DialogTitle } from '@mui/material';
import Grid from '@mui/material/Grid2';
import { useForm } from '@tanstack/react-form';
import { useMutation } from '@tanstack/react-query';
import { z } from 'zod';
import { apiNewsCreate, apiNewsSave, NewsEntry } from '../../api/news.ts';
import { useUserFlashCtx } from '../../hooks/useUserFlashCtx.ts';
import { Heading } from '../Heading';
import { Buttons } from '../field/Buttons.tsx';
import { CheckboxSimple } from '../field/CheckboxSimple.tsx';
import { MarkdownField } from '../field/MarkdownField.tsx';
import { TextFieldSimple } from '../field/TextFieldSimple.tsx';

export const NewsEditModal = NiceModal.create(({ entry }: { entry?: NewsEntry }) => {
    const modal = useModal();
    const { sendError, sendFlash } = useUserFlashCtx();

    const mutation = useMutation({
        mutationKey: ['newsEdit'],
        mutationFn: async (values: { title: string; body_md: string; is_published: boolean }) => {
            if (entry?.news_id) {
                return await apiNewsSave({ ...entry, ...values });
            } else {
                return await apiNewsCreate(values.title, values.body_md, values.is_published);
            }
        },
        onSuccess: async (entry) => {
            modal.resolve(entry);
            sendFlash('success', 'News edited successfully.');
            await modal.hide();
        },
        onError: sendError
    });

    const { Field, Subscribe, handleSubmit, reset } = useForm({
        onSubmit: async ({ value }) => {
            mutation.mutate(value);
        },
        defaultValues: {
            title: entry?.title ?? '',
            body_md: entry?.body_md ?? '',
            is_published: entry?.is_published ?? false
        }
    });

    return (
        <Dialog {...muiDialogV5(modal)} fullWidth maxWidth={'sm'}>
            <form
                onSubmit={async (e) => {
                    e.preventDefault();
                    e.stopPropagation();
                    await handleSubmit();
                }}
            >
                <DialogTitle component={Heading} iconLeft={<PersonIcon />}>
                    News Editor
                </DialogTitle>
                <DialogContent>
                    <Grid container spacing={2}>
                        <Grid size={{ xs: 12 }}>
                            <Field
                                name={'title'}
                                children={(props) => {
                                    return <TextFieldSimple {...props} label={'Title'} />;
                                }}
                            />
                        </Grid>
                        <Grid size={{ xs: 12 }}>
                            <Field
                                name={'body_md'}
                                validators={{
                                    onChange: z.string().min(10).default('')
                                }}
                                children={(props) => {
                                    return <MarkdownField {...props} label={'Body'} />;
                                }}
                            />
                        </Grid>
                        <Grid size={{ xs: 12 }}>
                            <Field
                                name={'is_published'}
                                children={(props) => {
                                    return <CheckboxSimple {...props} label={'Is Published'} />;
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
