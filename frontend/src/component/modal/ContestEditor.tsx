import React from 'react';
import { useFormik } from 'formik';
import {
    Dialog,
    DialogActions,
    DialogContent,
    DialogTitle
} from '@mui/material';
import GavelIcon from '@mui/icons-material/Gavel';
import Stack from '@mui/material/Stack';
import * as yup from 'yup';
import {
    boolDefinedValidator,
    dateAfterValidator,
    dateDefinedValidator,
    mimeTypesValidator,
    minNumberValidator,
    minStringValidator,
    permissionValidator
} from '../formik/Validator';
import TextField from '@mui/material/TextField';
import { DateTimePicker } from '@mui/x-date-pickers';
import { DateTimeValidationError } from '@mui/x-date-pickers';
import { BaseFormikInputProps } from '../formik/SteamIdField';
import { Heading } from '../Heading';
import NiceModal, { useModal, muiDialogV5 } from '@ebay/nice-modal-react';
import { PermissionLevel, useContest } from '../../api';
import { useUserFlashCtx } from '../../contexts/UserFlashCtx';
import { apiContestSave } from '../../api';
import { logErr } from '../../util/errors';
import { LoadingSpinner } from '../LoadingSpinner';

interface ContestEditorFormValues {
    contest_id: string;
    title: string;
    description: string;
    public: boolean;
    date_start: Date;
    date_end: Date;
    max_submissions: number;
    media_types: string[];
    voting: boolean;
    min_permission_level: PermissionLevel;
    down_votes: boolean;
}

const validationSchema = yup.object({
    title: minStringValidator('Title', 4),
    description: minStringValidator('Description', 1),
    public: boolDefinedValidator('Public'),
    date_start: dateDefinedValidator('Start date'),
    date_end: dateAfterValidator('date_start', 'End date'),
    max_submissions: minNumberValidator('Submissions', 1),
    media_types: mimeTypesValidator(),
    voting: boolDefinedValidator('Voting'),
    down_votes: boolDefinedValidator('Down votes'),
    min_permission_level: permissionValidator()
});

export const ContestEditor = NiceModal.create(
    ({ contest_id }: { contest_id?: number }) => {
        const { loading, contest } = useContest(contest_id);
        const modal = useModal();
        const { sendFlash } = useUserFlashCtx();

        const defaultStartDate = new Date();
        const defaultEndDate = new Date();
        defaultEndDate.setDate(defaultStartDate.getDate() + 1);

        const formik = useFormik<ContestEditorFormValues>({
            initialValues: {
                contest_id: contest?.contest_id ?? '',
                title: contest?.title ?? '',
                description: contest?.description ?? '',
                public: contest?.public ?? false,
                date_start: contest?.date_start ?? defaultStartDate,
                date_end: contest?.date_end ?? defaultEndDate,
                max_submissions: contest?.max_submissions ?? 1,
                media_types: contest?.media_types ?? [],
                voting: contest?.voting ?? false,
                down_votes: contest?.down_votes ?? false,
                min_permission_level:
                    contest?.min_permission_level ?? PermissionLevel.User
            },
            validateOnBlur: true,
            validateOnChange: false,
            onReset: () => {
                alert('reset!');
            },
            validationSchema: validationSchema,
            onSubmit: async (values) => {
                try {
                    const contest = await apiContestSave({
                        contest_id: values.contest_id,
                        date_start: values.date_start,
                        date_end: values.date_end,
                        description: values.description,
                        title: values.title,
                        voting: values.voting,
                        down_votes: values.down_votes,
                        max_submissions: values.max_submissions,
                        media_types: values.media_types,
                        public: values.public,
                        min_permission_level: values.min_permission_level,
                        deleted: false,
                        num_entries: 0
                    });
                    sendFlash(
                        'success',
                        `Contest created successfully (${contest.contest_id}`
                    );
                } catch (e) {
                    logErr(e);
                    sendFlash('error', 'Error saving contest');
                }
            }
        });

        const formId = 'contestEditorForm';

        return (
            <form onSubmit={formik.handleSubmit} id={formId}>
                <Dialog fullWidth {...muiDialogV5(modal)}>
                    <DialogTitle
                        component={Heading}
                        iconLeft={loading ? <LoadingSpinner /> : <GavelIcon />}
                    >
                        {`${
                            formik.values.contest_id.length > 0
                                ? 'Edit'
                                : 'Create'
                        } A Contest`}
                    </DialogTitle>

                    <DialogContent>
                        <Stack spacing={2}>
                            <TitleField
                                formik={formik}
                                fullWidth
                                isReadOnly={false}
                            />
                            <DescriptionField
                                formik={formik}
                                fullWidth
                                isReadOnly={false}
                            />
                            <Stack direction={'row'}>
                                <DateStartField
                                    formik={formik}
                                    fullWidth
                                    isReadOnly={false}
                                />
                                <DateEndField
                                    formik={formik}
                                    fullWidth
                                    isReadOnly={false}
                                />
                            </Stack>
                        </Stack>
                    </DialogContent>
                    <DialogActions></DialogActions>
                </Dialog>
            </form>
        );
    }
);

interface DateEndInputValue {
    date_end: Date;
}

const DateEndField = ({
    formik,
    isReadOnly
}: BaseFormikInputProps<DateEndInputValue>) => {
    return (
        <DateTimePicker
            disabled={isReadOnly ?? false}
            label={'End date'}
            value={formik.values.date_end}
            onChange={formik.handleChange}
        />
    );
};
interface DateStartInputValue {
    date_start: Date;
}

const DateStartField = ({
    formik,
    isReadOnly
}: BaseFormikInputProps<DateStartInputValue>) => {
    const [error, setError] = React.useState<DateTimeValidationError | null>(
        null
    );

    return (
        <DateTimePicker
            disabled={isReadOnly ?? false}
            onError={(newError) => setError(newError)}
            slotProps={{
                textField: {
                    helperText: error
                }
            }}
            label={'Start date'}
            value={formik.values.date_start}
            onChange={formik.handleChange}
        />
    );
};

interface TitleInputValue {
    title: string;
}

const TitleField = ({
    formik,
    isReadOnly
}: BaseFormikInputProps<TitleInputValue>) => {
    return (
        <TextField
            fullWidth
            disabled={isReadOnly ?? false}
            name={'title'}
            id={'title'}
            label={'Title'}
            value={formik.values.title}
            onChange={formik.handleChange}
            error={formik.touched.title && Boolean(formik.errors.title)}
            helperText={formik.touched.title && formik.errors.title}
        />
    );
};

interface DescriptionInputValue {
    description: string;
}

const DescriptionField = ({
    formik,
    isReadOnly
}: BaseFormikInputProps<DescriptionInputValue>) => {
    return (
        <TextField
            fullWidth
            multiline
            minRows={10}
            disabled={isReadOnly ?? false}
            name={'description'}
            id={'description'}
            label={'Description'}
            value={formik.values.description}
            onChange={formik.handleChange}
            error={
                formik.touched.description && Boolean(formik.errors.description)
            }
            helperText={formik.touched.description && formik.errors.description}
        />
    );
};
