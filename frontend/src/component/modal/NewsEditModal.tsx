import NiceModal, { muiDialogV5, useModal } from '@ebay/nice-modal-react';
import PersonIcon from '@mui/icons-material/Person';
import { Dialog, DialogActions, DialogContent, DialogTitle } from '@mui/material';
import Grid from '@mui/material/Unstable_Grid2';
import { useForm } from '@tanstack/react-form';
import { useMutation } from '@tanstack/react-query';
import { zodValidator } from '@tanstack/zod-form-adapter';
import { apiNewsCreate, apiNewsSave, NewsEntry } from '../../api/news.ts';
import { Heading } from '../Heading';
import { Buttons } from '../field/Buttons.tsx';
import { CheckboxSimple } from '../field/CheckboxSimple.tsx';
import { MarkdownField } from '../field/MarkdownField.tsx';
import { TextFieldSimple } from '../field/TextFieldSimple.tsx';

export const NewsEditModal = NiceModal.create(({ entry }: { entry?: NewsEntry }) => {
    const modal = useModal();

    const mutation = useMutation({
        mutationKey: ['newsEdit'],
        mutationFn: async (values: { title: string; body_md: string; is_published: boolean }) => {
            try {
                if (entry?.news_id) {
                    modal.resolve(await apiNewsSave({ ...entry, ...values }));
                } else {
                    modal.resolve(await apiNewsCreate(values.title, values.body_md, values.is_published));
                }
            } catch (e) {
                modal.reject(e);
            }
            await modal.hide();
        }
    });

    const { Field, Subscribe, handleSubmit, reset } = useForm({
        onSubmit: async ({ value }) => {
            mutation.mutate(value);
        },
        validatorAdapter: zodValidator,
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
                        <Grid xs={12}>
                            <Field
                                name={'title'}
                                children={(props) => {
                                    return <TextFieldSimple {...props} label={'Title'} />;
                                }}
                            />
                        </Grid>
                        <Grid xs={12}>
                            <Field
                                name={'body_md'}
                                children={(props) => {
                                    return <MarkdownField {...props} label={'Body'} />;
                                }}
                            />
                        </Grid>
                        <Grid xs={12}>
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
                        <Grid xs={12} mdOffset="auto">
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
