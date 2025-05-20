import { useMemo } from 'react';
import NiceModal, { muiDialogV5, useModal } from '@ebay/nice-modal-react';
import { Dialog, DialogActions, DialogContent, DialogTitle } from '@mui/material';
import ButtonGroup from '@mui/material/ButtonGroup';
import Grid from '@mui/material/Grid';
import MenuItem from '@mui/material/MenuItem';
import { useMutation } from '@tanstack/react-query';
import { z } from 'zod/v4';
import { apiCreateForum, apiSaveForum } from '../../api/forum.ts';
import { useAppForm } from '../../contexts/formContext.tsx';
import { useUserFlashCtx } from '../../hooks/useUserFlashCtx.ts';
import { Forum, ForumCategory } from '../../schema/forum.ts';
import {
    PermissionLevel,
    PermissionLevelCollection,
    PermissionLevelEnum,
    permissionLevelString
} from '../../schema/people.ts';

type ForumEditorValues = {
    forum_category_id: number;
    title: string;
    description: string;
    ordering: string;
    permission_level: PermissionLevelEnum;
};

export const ForumForumEditorModal = NiceModal.create(
    ({ forum, categories }: { forum?: Forum; categories: ForumCategory[] }) => {
        const modal = useModal();
        const { sendError } = useUserFlashCtx();

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
            onError: sendError
        });

        const defaultCategory = forum?.forum_category_id
            ? (categories.find((value) => value.forum_category_id == forum.forum_category_id)?.forum_category_id ??
              categories[0].forum_category_id)
            : categories[0].forum_category_id;

        const form = useAppForm({
            onSubmit: async ({ value }) => {
                mutation.mutate({ ...value });
            },
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
                        await form.handleSubmit();
                    }}
                >
                    <DialogTitle>Category Editor</DialogTitle>

                    <DialogContent>
                        <Grid container spacing={2}>
                            <Grid size={{ xs: 12 }}>
                                <form.AppField
                                    name={'forum_category_id'}
                                    children={(field) => {
                                        return (
                                            <field.SelectField
                                                label={'Category'}
                                                items={catIds}
                                                renderItem={(catId) => {
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
                            <Grid size={{ xs: 12 }}>
                                <form.AppField
                                    name={'title'}
                                    validators={{
                                        onChange: z.string().min(1)
                                    }}
                                    children={(field) => {
                                        return <field.TextField label={'Title'} />;
                                    }}
                                />
                            </Grid>
                            <Grid size={{ xs: 12 }}>
                                <form.AppField
                                    name={'description'}
                                    validators={{
                                        onChange: z.string().min(1)
                                    }}
                                    children={(field) => {
                                        return <field.TextField label={'Description'} rows={5} />;
                                    }}
                                />
                            </Grid>
                            <Grid size={{ xs: 12 }}>
                                <form.AppField
                                    name={'ordering'}
                                    validators={{
                                        onChange: z.string().min(1)
                                    }}
                                    children={(field) => {
                                        return <field.TextField label={'Order'} />;
                                    }}
                                />
                            </Grid>
                            <Grid size={{ xs: 12 }}>
                                <form.AppField
                                    name={'permission_level'}
                                    validators={{
                                        onChange: z.nativeEnum(PermissionLevel)
                                    }}
                                    children={(field) => {
                                        return (
                                            <field.SelectField
                                                label={'Permissions Required'}
                                                items={PermissionLevelCollection}
                                                renderItem={(pl) => {
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
                            <Grid size={{ xs: 12 }}>
                                <form.AppForm>
                                    <ButtonGroup>
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
    }
);
