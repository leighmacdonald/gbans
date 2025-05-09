import NiceModal, { muiDialogV5, useModal } from '@ebay/nice-modal-react';
import PersonIcon from '@mui/icons-material/Person';
import { Dialog, DialogActions, DialogContent, DialogTitle } from '@mui/material';
import Grid from '@mui/material/Grid';
import { useMutation } from '@tanstack/react-query';
import { z } from 'zod';
import { apiNewsCreate, apiNewsSave, NewsEntry } from '../../api/news.ts';
import { useAppForm } from '../../contexts/formContext.tsx';
import { useUserFlashCtx } from '../../hooks/useUserFlashCtx.ts';
import { Heading } from '../Heading';

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

    const form = useAppForm({
        onSubmit: async ({ value }) => {
            mutation.mutate(value);
        },
        defaultValues: {
            title: entry?.title ?? '',
            body_md: entry?.body_md ?? '',
            is_published: entry?.is_published ?? false
        },
        validators: {
            onSubmit: z.object({
                body_md: z.string().min(10),
                title: z.string().min(4),
                is_published: z.boolean()
            })
        }
    });

    return (
        <Dialog {...muiDialogV5(modal)} fullWidth maxWidth={'sm'}>
            <form
                onSubmit={async (e) => {
                    e.preventDefault();
                    e.stopPropagation();
                    await form.handleSubmit();
                }}
            >
                <DialogTitle component={Heading} iconLeft={<PersonIcon />}>
                    News Editor
                </DialogTitle>
                <DialogContent>
                    <Grid container spacing={2}>
                        <Grid size={{ xs: 12 }}>
                            <form.AppField
                                name={'title'}
                                children={(field) => {
                                    return <field.TextField label={'Title'} />;
                                }}
                            />
                        </Grid>
                        <Grid size={{ xs: 12 }}>
                            <form.AppField
                                name={'body_md'}
                                children={(field) => {
                                    return <field.MarkdownField label={'Body'} />;
                                }}
                            />
                        </Grid>
                        <Grid size={{ xs: 12 }}>
                            <form.AppField
                                name={'is_published'}
                                children={(field) => {
                                    return <field.CheckboxField label={'Is Published'} />;
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
