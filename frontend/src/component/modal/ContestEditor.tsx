import NiceModal, { muiDialogV5, useModal } from '@ebay/nice-modal-react';
import EmojiEventsIcon from '@mui/icons-material/EmojiEvents';
import { Dialog, DialogActions, DialogContent, DialogTitle } from '@mui/material';
import ButtonGroup from '@mui/material/ButtonGroup';
import Grid from '@mui/material/Grid';
import MenuItem from '@mui/material/MenuItem';
import { useMutation } from '@tanstack/react-query';
import { z } from 'zod';
import {
    apiContestSave,
    Contest,
    EmptyUUID,
    PermissionLevel,
    PermissionLevelCollection,
    permissionLevelString
} from '../../api';
import { useAppForm } from '../../contexts/formContext.tsx';
import { useUserFlashCtx } from '../../hooks/useUserFlashCtx.ts';
import { Heading } from '../Heading';

const schema = z.object({
    title: z.string().min(2),
    description: z.string().min(10),
    hide_submissions: z.boolean(),
    public: z.boolean(),
    date_start: z.date(),
    date_end: z.date(),
    max_submissions: z.number(),
    media_types: z.string().refine((arg) => {
        if (arg == '') {
            return true;
        }

        const parts = arg?.split(',');
        const matches = parts.filter((p) => p.match(/^\S+\/\S+$/));
        return matches.length == parts.length;
    }),
    voting: z.boolean(),
    min_permission_level: z.nativeEnum(PermissionLevel),
    down_votes: z.boolean(),
    deleted: z.boolean()
});

export const ContestEditor = NiceModal.create(({ contest }: { contest?: Contest }) => {
    const modal = useModal();
    const { sendError } = useUserFlashCtx();

    const mutation = useMutation({
        mutationKey: ['adminContest'],
        mutationFn: async (values: z.input<typeof schema>) => {
            return await apiContestSave({
                contest_id: contest?.contest_id ?? EmptyUUID,
                date_start: values.date_start,
                date_end: values.date_end,
                description: values.description,
                hide_submissions: values.hide_submissions,
                title: values.title,
                voting: values.voting,
                down_votes: values.down_votes,
                max_submissions: values.max_submissions,
                media_types: values.media_types,
                public: values.public,
                min_permission_level: values.min_permission_level,
                deleted: values.deleted ?? false,
                num_entries: 0,
                updated_on: new Date(),
                created_on: new Date()
            });
        },
        onSuccess: async (contest) => {
            modal.resolve(contest);
            await modal.hide();
        },
        onError: sendError
    });

    const form = useAppForm({
        onSubmit: async ({ value }) => {
            mutation.mutate(value);
        },
        defaultValues: {
            date_start: contest?.date_start ?? new Date(),
            date_end: contest?.date_end ?? new Date(),
            description: contest?.description ?? '',
            hide_submissions: contest?.hide_submissions ?? false,
            title: contest?.title ?? '',
            voting: contest?.voting ?? true,
            down_votes: contest?.down_votes ?? true,
            max_submissions: contest?.max_submissions ?? 1,
            media_types: contest?.media_types ?? '',
            public: contest?.public ?? true,
            min_permission_level: contest?.min_permission_level ?? PermissionLevel.User,
            deleted: contest?.deleted ?? false
        },
        validators: {
            onSubmit: schema
        }
    });

    return (
        <Dialog fullWidth {...muiDialogV5(modal)}>
            <form
                onSubmit={async (e) => {
                    e.preventDefault();
                    e.stopPropagation();
                    await form.handleSubmit();
                }}
            >
                <DialogTitle component={Heading} iconLeft={<EmojiEventsIcon />}>
                    {`${contest?.contest_id == EmptyUUID ? 'Create' : 'Edit'} A Contest`}
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
                                name={'description'}
                                children={(field) => {
                                    return <field.MarkdownField label={'Description'} rows={10} />;
                                }}
                            />
                        </Grid>
                        <Grid size={{ xs: 4 }}>
                            <form.AppField
                                name={'public'}
                                children={(field) => {
                                    return <field.CheckboxField label={'Public'} />;
                                }}
                            />
                        </Grid>
                        <Grid size={{ xs: 4 }}>
                            <form.AppField
                                name={'hide_submissions'}
                                children={(field) => {
                                    return <field.CheckboxField label={'Hide Submissions'} />;
                                }}
                            />
                        </Grid>
                        <Grid size={{ xs: 4 }}>
                            <form.AppField
                                name={'max_submissions'}
                                children={(field) => {
                                    return <field.TextField label={'Max Submissions'} />;
                                }}
                            />
                        </Grid>
                        <Grid size={{ xs: 6 }}>
                            <form.AppField
                                name={'min_permission_level'}
                                children={(field) => {
                                    return (
                                        <field.SelectField
                                            label={'Min Permissions'}
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
                        <Grid size={{ xs: 6 }}>
                            <form.AppField
                                name={'voting'}
                                children={(field) => {
                                    return <field.CheckboxField label={'Voting Enabled'} />;
                                }}
                            />
                        </Grid>
                        <Grid size={{ xs: 6 }}>
                            <form.AppField
                                name={'down_votes'}
                                children={(field) => {
                                    return <field.CheckboxField label={'Downvotes Enabled'} />;
                                }}
                            />
                        </Grid>
                        <Grid size={{ xs: 6 }}>
                            <form.AppField
                                name={'date_start'}
                                children={(field) => {
                                    return <field.DateTimeField label={'Start Date'} />;
                                }}
                            />
                        </Grid>
                        <Grid size={{ xs: 6 }}>
                            <form.AppField
                                name={'date_end'}
                                children={(field) => {
                                    return <field.DateTimeField label={'End Date'} />;
                                }}
                            />
                        </Grid>
                        <Grid size={{ xs: 6 }}>
                            <form.AppField
                                name={'media_types'}
                                children={(field) => {
                                    return (
                                        <field.TextField
                                            label={'Allowed Mime Types'}
                                            helperText={
                                                'A comma separated list of acceptable mime types. If empty, all types are allowed.'
                                            }
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
});
