import NiceModal, { muiDialogV5, useModal } from '@ebay/nice-modal-react';
import EmojiEventsIcon from '@mui/icons-material/EmojiEvents';
import { Dialog, DialogActions, DialogContent, DialogTitle } from '@mui/material';
import MenuItem from '@mui/material/MenuItem';
import Grid from '@mui/material/Unstable_Grid2';
import { useForm } from '@tanstack/react-form';
import { useMutation } from '@tanstack/react-query';
import { zodValidator } from '@tanstack/zod-form-adapter';
import { parseISO } from 'date-fns';
import { z } from 'zod';
import {
    apiContestSave,
    Contest,
    EmptyUUID,
    PermissionLevel,
    PermissionLevelCollection,
    permissionLevelString
} from '../../api';
import { useUserFlashCtx } from '../../hooks/useUserFlashCtx.ts';
import { numberStringValidator } from '../../util/validator/numberStringValidator.ts';
import { Heading } from '../Heading';
import { Buttons } from '../field/Buttons.tsx';
import { CheckboxSimple } from '../field/CheckboxSimple.tsx';
import { DateTimeSimple } from '../field/DateTimeSimple.tsx';
import { MarkdownField } from '../field/MarkdownField.tsx';
import { SelectFieldSimple } from '../field/SelectFieldSimple.tsx';
import { TextFieldSimple } from '../field/TextFieldSimple.tsx';

type ContestEditorFormValues = {
    title: string;
    description: string;
    hide_submissions: boolean;
    public: boolean;
    date_start: string;
    date_end: string;
    max_submissions: string;
    media_types: string;
    voting: boolean;
    min_permission_level: PermissionLevel;
    down_votes: boolean;
};

// const validationSchema = yup.object({
//     title: minStringValidator('Title', 4),
//     description: minStringValidator('Description', 1),
//     public: boolDefinedValidator('Public'),
//     date_start: dateDefinedValidator('date_start'),
//     date_end: dateAfterValidator('date_start', 'End date'),
//     max_submissions: numberValidator('Submissions'),
//     media_types: mimeTypesValidator(),
//     voting: boolDefinedValidator('Voting'),
//     hide_submissions: boolDefinedValidator('Hide Submissions'),
//     down_votes: boolDefinedValidator('Down votes'),
//     min_permission_level: permissionValidator()
// });

export const ContestEditor = NiceModal.create(({ contest }: { contest?: Contest }) => {
    const modal = useModal();
    const { sendError } = useUserFlashCtx();

    const mutation = useMutation({
        mutationKey: ['adminContest'],
        mutationFn: async (values: ContestEditorFormValues) => {
            return await apiContestSave({
                contest_id: contest?.contest_id ?? EmptyUUID,
                date_start: parseISO(values.date_start),
                date_end: parseISO(values.date_end),
                description: values.description,
                hide_submissions: values.hide_submissions,
                title: values.title,
                voting: values.voting,
                down_votes: values.down_votes,
                max_submissions: Number(values.max_submissions),
                media_types: values.media_types,
                public: values.public,
                min_permission_level: values.min_permission_level,
                deleted: false,
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

    const { Field, Subscribe, handleSubmit, reset } = useForm({
        onSubmit: async ({ value }) => {
            mutation.mutate(value);
        },
        validatorAdapter: zodValidator,
        defaultValues: {
            date_start: contest?.date_start.toISOString() ?? '',
            date_end: contest ? contest.date_end.toISOString() : '',
            description: contest ? contest.description : '',
            hide_submissions: contest ? contest.hide_submissions : false,
            title: contest ? contest.title : '',
            voting: contest ? contest.voting : true,
            down_votes: contest ? contest.down_votes : true,
            max_submissions: contest ? String(contest.max_submissions) : '1',
            media_types: contest ? contest.media_types : '',
            public: contest ? contest.public : true,
            min_permission_level: contest ? contest.min_permission_level : PermissionLevel.User,
            deleted: contest ? contest.deleted : false,
            num_entries: 0,
            updated_on: new Date(),
            created_on: new Date()
        }
    });

    return (
        <Dialog fullWidth {...muiDialogV5(modal)}>
            <form
                onSubmit={async (e) => {
                    e.preventDefault();
                    e.stopPropagation();
                    await handleSubmit();
                }}
            >
                <DialogTitle component={Heading} iconLeft={<EmojiEventsIcon />}>
                    {`${contest?.contest_id == EmptyUUID ? 'Create' : 'Edit'} A Contest`}
                </DialogTitle>

                <DialogContent>
                    <Grid container spacing={2}>
                        <Grid xs={12}>
                            <Field
                                name={'title'}
                                validators={{
                                    onChange: z.string().min(5)
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
                                    onChange: z.string().min(5)
                                }}
                                children={(props) => {
                                    return (
                                        <MarkdownField {...props} label={'Description'} multiline={true} rows={10} />
                                    );
                                }}
                            />
                        </Grid>
                        <Grid xs={4}>
                            <Field
                                name={'public'}
                                validators={{
                                    onChange: z.boolean()
                                }}
                                children={(props) => {
                                    return <CheckboxSimple {...props} label={'Public'} />;
                                }}
                            />
                        </Grid>
                        <Grid xs={4}>
                            <Field
                                name={'hide_submissions'}
                                validators={{
                                    onChange: z.boolean()
                                }}
                                children={(props) => {
                                    return <CheckboxSimple {...props} label={'Hide Submissions'} />;
                                }}
                            />
                        </Grid>
                        <Grid xs={6}>
                            <Field
                                name={'max_submissions'}
                                validators={{
                                    onChange: z.string().transform(numberStringValidator(1, 10))
                                }}
                                children={(props) => {
                                    return <TextFieldSimple {...props} label={'Max Submissions'} />;
                                }}
                            />
                        </Grid>
                        <Grid xs={6}>
                            <Field
                                name={'min_permission_level'}
                                children={(props) => {
                                    return (
                                        <SelectFieldSimple
                                            {...props}
                                            label={'Min Permissions'}
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
                        <Grid xs={6}>
                            <Field
                                name={'voting'}
                                validators={{
                                    onChange: z.boolean()
                                }}
                                children={(props) => {
                                    return <CheckboxSimple {...props} label={'Voting Enabled'} />;
                                }}
                            />
                        </Grid>
                        <Grid xs={6}>
                            <Field
                                name={'down_votes'}
                                validators={{
                                    onChange: z.boolean()
                                }}
                                children={(props) => {
                                    return <CheckboxSimple {...props} label={'Downvotes Enabled'} />;
                                }}
                            />
                        </Grid>
                        <Grid xs={6}>
                            <Field
                                name={'date_start'}
                                children={(props) => {
                                    return <DateTimeSimple {...props} label={'Custom Expire Date'} />;
                                }}
                            />
                        </Grid>
                        <Grid xs={6}>
                            <Field
                                name={'date_end'}
                                children={(props) => {
                                    return <DateTimeSimple {...props} label={'Custom Expire Date'} />;
                                }}
                            />
                        </Grid>
                        <Grid xs={6}>
                            <Field
                                name={'media_types'}
                                validators={{
                                    onChange: z.string().refine((arg) => {
                                        if (arg == '') {
                                            return true;
                                        }

                                        const parts = arg?.split(',');
                                        const matches = parts.filter((p) => p.match(/^\S+\/\S+$/));
                                        return matches.length == parts.length;
                                    })
                                }}
                                children={(props) => {
                                    return <TextFieldSimple {...props} label={'Allowed Mime Types'} />;
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
