import { useMemo } from 'react';
import NiceModal, { muiDialogV5, useModal } from '@ebay/nice-modal-react';
import { Dialog, DialogActions, DialogContent, DialogTitle } from '@mui/material';
import MenuItem from '@mui/material/MenuItem';
import Grid from '@mui/material/Unstable_Grid2';
import { useForm } from '@tanstack/react-form';
import { useMutation } from '@tanstack/react-query';
import { zodValidator } from '@tanstack/zod-form-adapter';
import { z } from 'zod';
import { PermissionLevel, PermissionLevelCollection, permissionLevelString } from '../../api';
import { apiCreateForum, apiSaveForum, Forum, ForumCategory } from '../../api/forum.ts';
import { useUserFlashCtx } from '../../hooks/useUserFlashCtx.ts';
import { Buttons } from '../field/Buttons.tsx';
import { SelectFieldSimple } from '../field/SelectFieldSimple.tsx';
import { TextFieldSimple } from '../field/TextFieldSimple.tsx';

type ForumEditorValues = {
    forum_category_id: number;
    title: string;
    description: string;
    ordering: string;
    permission_level: PermissionLevel;
};

export const ForumForumEditorModal = NiceModal.create(
    ({ forum, categories }: { forum?: Forum; categories: ForumCategory[] }) => {
        const modal = useModal();
        const { sendFlash } = useUserFlashCtx();

        const mutation = useMutation({
            mutationKey: ['forumCategory'],
            mutationFn: async (values: ForumEditorValues) => {
                if (forum?.forum_id) {
                    return await apiSaveForum(
                        forum.forum_id,
                        Number(values.forum_category_id),
                        values.title,
                        values.description,
                        Number(values.ordering),
                        values.permission_level
                    );
                } else {
                    return await apiCreateForum(
                        Number(values.forum_category_id),
                        values.title,
                        values.description,
                        Number(values.ordering),
                        values.permission_level
                    );
                }
            },
            onSuccess: async (forum: Forum) => {
                modal.resolve(forum);
                await modal.hide();
            },
            onError: (error) => {
                sendFlash('error', `${error}`);
            }
        });

        const defaultCategory = forum?.forum_category_id
            ? categories.find((value) => value.forum_category_id == forum.forum_category_id)?.forum_category_id ??
              categories[0].forum_category_id
            : categories[0].forum_category_id;

        const { Field, Subscribe, handleSubmit, reset } = useForm({
            onSubmit: async ({ value }) => {
                mutation.mutate({ ...value });
            },
            validatorAdapter: zodValidator,
            defaultValues: {
                forum_category_id: defaultCategory,
                title: forum?.title ?? '',
                description: forum?.description ?? '',
                ordering: forum?.ordering ? String(forum?.ordering) : '0',
                permission_level: forum?.permission_level ?? PermissionLevel.User
            }
        });

        const catIds = useMemo(() => {
            return categories.map((c) => c.forum_category_id);
        }, [categories]);

        return (
            <Dialog {...muiDialogV5(modal)} fullWidth maxWidth={'lg'}>
                <form
                    onSubmit={async (e) => {
                        e.preventDefault();
                        e.stopPropagation();
                        await handleSubmit();
                    }}
                >
                    <DialogTitle>Category Editor</DialogTitle>

                    <DialogContent>
                        <Grid container spacing={2}>
                            <Grid xs={12}>
                                <Field
                                    name={'forum_category_id'}
                                    children={(props) => {
                                        return (
                                            <SelectFieldSimple
                                                {...props}
                                                label={'Category'}
                                                fullwidth={true}
                                                items={catIds}
                                                renderMenu={(catId) => {
                                                    return (
                                                        <MenuItem value={catId} key={`cat-${catId}`}>
                                                            {categories.find((c) => c.forum_category_id == catId)
                                                                ?.title ?? ''}
                                                        </MenuItem>
                                                    );
                                                }}
                                            />
                                        );
                                    }}
                                />
                            </Grid>
                            <Grid xs={12}>
                                <Field
                                    name={'title'}
                                    validators={{
                                        onChange: z.string().min(1)
                                    }}
                                    children={(props) => {
                                        return <TextFieldSimple {...props} label={'Title'} />;
                                    }}
                                />
                            </Grid>
                            <Grid xs={12}>
                                <Field
                                    name={'description'}
                                    validators={{
                                        onChange: z.string().min(1)
                                    }}
                                    children={(props) => {
                                        return <TextFieldSimple {...props} label={'Description'} rows={5} />;
                                    }}
                                />
                            </Grid>
                            <Grid xs={12}>
                                <Field
                                    name={'ordering'}
                                    validators={{
                                        onChange: z.string().min(1)
                                    }}
                                    children={(props) => {
                                        return <TextFieldSimple {...props} label={'Order'} />;
                                    }}
                                />
                            </Grid>
                            <Grid xs={12}>
                                <Field
                                    name={'permission_level'}
                                    validators={{
                                        onChange: z.nativeEnum(PermissionLevel)
                                    }}
                                    children={(props) => {
                                        return (
                                            <SelectFieldSimple
                                                {...props}
                                                label={'Permissions Required'}
                                                fullwidth={true}
                                                items={PermissionLevelCollection}
                                                renderMenu={(pl) => {
                                                    return (
                                                        <MenuItem value={pl} key={`pl-${pl}`}>
                                                            {permissionLevelString(pl)}
                                                        </MenuItem>
                                                    );
                                                }}
                                            />
                                        );
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
                                                closeLabel={'Cancel'}
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
