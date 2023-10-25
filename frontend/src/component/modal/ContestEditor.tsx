import React, { useMemo, useState } from 'react';
import NiceModal, { useModal, muiDialogV5 } from '@ebay/nice-modal-react';
import EmojiEventsIcon from '@mui/icons-material/EmojiEvents';
import {
    Dialog,
    DialogActions,
    DialogContent,
    DialogTitle
} from '@mui/material';
import Checkbox from '@mui/material/Checkbox';
import FormControl from '@mui/material/FormControl';
import FormControlLabel from '@mui/material/FormControlLabel';
import FormGroup from '@mui/material/FormGroup';
import FormHelperText from '@mui/material/FormHelperText';
import InputLabel from '@mui/material/InputLabel';
import MenuItem from '@mui/material/MenuItem';
import Select from '@mui/material/Select';
import Stack from '@mui/material/Stack';
import TextField from '@mui/material/TextField';
import { DateTimePicker } from '@mui/x-date-pickers';
import { DateTimeValidationError } from '@mui/x-date-pickers';
import { useFormik } from 'formik';
import * as yup from 'yup';
import { EmptyUUID, PermissionLevel, useContest } from '../../api';
import { apiContestSave } from '../../api';
import { useUserFlashCtx } from '../../contexts/UserFlashCtx';
import { logErr } from '../../util/errors';
import { Heading } from '../Heading';
import { LoadingSpinner } from '../LoadingSpinner';
import { BaseFormikInputProps } from '../formik/SteamIdField';
import {
    boolDefinedValidator,
    dateAfterValidator,
    dateDefinedValidator,
    mimeTypesValidator,
    minStringValidator,
    numberValidator,
    permissionValidator
} from '../formik/Validator';
import { CancelButton, ResetButton, SaveButton } from './Buttons';

interface ContestEditorFormValues {
    contest_id: string;
    title: string;
    description: string;
    hide_submissions: boolean;
    public: boolean;
    date_start: Date;
    date_end: Date;
    max_submissions: number;
    media_types: string;
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
    max_submissions: numberValidator('Submissions'),
    media_types: mimeTypesValidator(),
    voting: boolDefinedValidator('Voting'),
    hide_submissions: boolDefinedValidator('Hide Submissions'),
    down_votes: boolDefinedValidator('Down votes'),
    min_permission_level: permissionValidator()
});

export const ContestEditor = NiceModal.create(
    ({ contest_id }: { contest_id?: string }) => {
        const { loading, contest } = useContest(contest_id);
        const modal = useModal();
        const { sendFlash } = useUserFlashCtx();

        const defaultStartDate = useMemo(() => new Date(), []);

        const defaultEndDate = useMemo(() => {
            const endDate = new Date();
            endDate.setDate(defaultStartDate.getDate() + 1);
            return endDate;
        }, [defaultStartDate]);

        const formik = useFormik<ContestEditorFormValues>({
            initialValues: {
                contest_id: contest?.contest_id ?? EmptyUUID,
                title: contest?.title ?? '',
                description: contest?.description ?? '',
                public: contest?.public ?? false,
                date_start: contest?.date_start ?? defaultStartDate,
                date_end: contest?.date_end ?? defaultEndDate,
                hide_submissions: contest?.hide_submissions ?? false,
                max_submissions: contest?.max_submissions ?? 1,
                media_types: contest?.media_types ?? '',
                voting: contest?.voting ?? false,
                down_votes: contest?.down_votes ?? false,
                min_permission_level:
                    contest?.min_permission_level ?? PermissionLevel.User
            },
            validateOnBlur: false,
            validateOnChange: false,
            validationSchema: validationSchema,
            enableReinitialize: true,
            onSubmit: async (values) => {
                console.log('submitted');
                try {
                    const contest = await apiContestSave({
                        contest_id:
                            values.contest_id != ''
                                ? values.contest_id
                                : EmptyUUID,
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
                        deleted: false,
                        num_entries: 0
                    });
                    sendFlash(
                        'success',
                        `Contest created successfully (${contest.contest_id}`
                    );
                    await modal.hide();
                } catch (e) {
                    logErr(e);
                    sendFlash('error', 'Error saving contest');
                }
            }
        });

        // const onSave = useCallback(async () => {
        //     console.log('submitting');
        //     await formik.submitForm();
        //     console.log('submitted');
        // }, [formik]);

        const formId = 'contestEditorForm';

        return (
            <form onSubmit={formik.handleSubmit} id={formId}>
                <Dialog fullWidth {...muiDialogV5(modal)}>
                    <DialogTitle
                        component={Heading}
                        iconLeft={
                            loading ? <LoadingSpinner /> : <EmojiEventsIcon />
                        }
                    >
                        {`${
                            formik.values.contest_id == EmptyUUID
                                ? 'Create'
                                : 'Edit'
                        } A Contest`}
                    </DialogTitle>

                    <DialogContent>
                        {loading ? (
                            <LoadingSpinner />
                        ) : (
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
                                <Stack direction={'row'} spacing={2}>
                                    <PublicField
                                        formik={formik}
                                        fullWidth
                                        isReadOnly={false}
                                    />
                                    <HideSubmissionsField
                                        formik={formik}
                                        fullWidth
                                        isReadOnly={false}
                                    />
                                    <MaxSubmissionsField
                                        formik={formik}
                                        fullWidth
                                        isReadOnly={false}
                                    />
                                    <MinPermissionLevelField
                                        formik={formik}
                                        fullWidth
                                        isReadOnly={false}
                                    />
                                </Stack>
                                <Stack direction={'row'} spacing={2}>
                                    <VotingField
                                        fullWidth
                                        formik={formik}
                                        isReadOnly={false}
                                    />

                                    <DownVotesField
                                        fullWidth
                                        formik={formik}
                                        isReadOnly={formik.values.voting}
                                    />
                                </Stack>

                                <Stack direction={'row'} spacing={2}>
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

                                <MimeTypeField
                                    formik={formik}
                                    fullWidth
                                    isReadOnly={false}
                                />
                            </Stack>
                        )}
                    </DialogContent>
                    <DialogActions>
                        <CancelButton onClick={modal.hide} />
                        <ResetButton onClick={formik.resetForm} />
                        <SaveButton onClick={formik.submitForm} />
                    </DialogActions>
                </Dialog>
            </form>
        );
    }
);

interface MaxSubmissionsInputValue {
    max_submissions: number;
}

const MaxSubmissionsField = ({
    formik,
    isReadOnly
}: BaseFormikInputProps<MaxSubmissionsInputValue>) => {
    return (
        <FormControl fullWidth>
            <InputLabel id="max-subs-select-label">
                Maximum Submissions Per User
            </InputLabel>
            <Select<number>
                name={`max-subs`}
                labelId={`max-subs-select-label`}
                id={`max-subs-selects`}
                disabled={isReadOnly ?? false}
                label={'Maximum Submissions Per User'}
                value={formik.values.max_submissions}
                onChange={(event) => {
                    console.log(event.target.value);
                    formik.values.max_submissions = event.target
                        .value as number;
                }}
            >
                {[-1, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10].map((c) => (
                    <MenuItem key={`max-subs-${c}`} value={c}>
                        {c}
                    </MenuItem>
                ))}
            </Select>
            <FormHelperText>-1 indicates unlimited</FormHelperText>
        </FormControl>
    );
};

interface MinPermissionLevelInputValue {
    min_permission_level: number;
}

const MinPermissionLevelField = ({
    formik,
    isReadOnly
}: BaseFormikInputProps<MinPermissionLevelInputValue>) => {
    return (
        <FormControl fullWidth>
            <InputLabel id="plevel-label">
                Minimum permissions required to submit
            </InputLabel>
            <Select<number>
                name={`plevel`}
                labelId={`plevel-label`}
                id={`plevel`}
                disabled={isReadOnly ?? false}
                label={'Minimum permissions required to submit'}
                value={formik.values.min_permission_level}
                onChange={formik.handleChange}
            >
                <MenuItem value={PermissionLevel.User}>Logged In User</MenuItem>
                <MenuItem value={PermissionLevel.Editor}>Editor</MenuItem>
                <MenuItem value={PermissionLevel.Moderator}>Moderator</MenuItem>
                <MenuItem value={PermissionLevel.Admin}>Admin</MenuItem>
            </Select>
        </FormControl>
    );
};

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
    const [error, setError] = useState<DateTimeValidationError | null>(null);

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

interface PublicFieldInputValue {
    public: boolean;
}

const PublicField = ({
    formik,
    isReadOnly
}: BaseFormikInputProps<PublicFieldInputValue>) => {
    return (
        <FormGroup>
            <FormControlLabel
                control={
                    <Checkbox
                        id={'public-cb'}
                        disabled={isReadOnly ?? false}
                        value={formik.values.public}
                        onChange={formik.handleChange}
                        onBlur={formik.handleBlur}
                    />
                }
                label="Public"
            />
        </FormGroup>
    );
};

interface HideSubmissionsFieldInputValue {
    hide_submissions: boolean;
}

const HideSubmissionsField = ({
    formik,
    isReadOnly
}: BaseFormikInputProps<HideSubmissionsFieldInputValue>) => {
    return (
        <FormGroup>
            <FormControlLabel
                control={
                    <Checkbox
                        id={'hide_submissions-cb'}
                        disabled={isReadOnly ?? false}
                        value={formik.values.hide_submissions}
                        onChange={formik.handleChange}
                        onBlur={formik.handleBlur}
                    />
                }
                label="Hide Submissions"
            />
        </FormGroup>
    );
};

interface VotingInputValue {
    voting: boolean;
}

const VotingField = ({
    formik,
    isReadOnly
}: BaseFormikInputProps<VotingInputValue>) => {
    return (
        <FormGroup>
            <FormControlLabel
                control={
                    <Checkbox
                        id={'voting-cb'}
                        disabled={isReadOnly ?? false}
                        value={formik.values.voting}
                        onChange={formik.handleChange}
                    />
                }
                label="Voting Allowed"
            />
        </FormGroup>
    );
};

interface DownVotesInputValue {
    down_votes: boolean;
}

const DownVotesField = ({
    formik,
    isReadOnly
}: BaseFormikInputProps<DownVotesInputValue>) => {
    return (
        <FormGroup>
            <FormControlLabel
                control={
                    <Checkbox
                        id={'down_votes-cb'}
                        disabled={isReadOnly ?? false}
                        value={formik.values.down_votes}
                        onChange={formik.handleChange}
                    />
                }
                label="Down Votes Allowed"
            />
        </FormGroup>
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
            onBlur={formik.handleBlur}
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
            onBlur={formik.handleBlur}
            error={
                formik.touched.description && Boolean(formik.errors.description)
            }
            helperText={formik.touched.description && formik.errors.description}
        />
    );
};

interface MimeTypeInputValue {
    media_types: string;
}

const MimeTypeField = ({
    formik,
    isReadOnly
}: BaseFormikInputProps<MimeTypeInputValue>) => {
    return (
        <TextField
            fullWidth
            disabled={isReadOnly ?? false}
            name={'media_types'}
            id={'media_types'}
            label={'Mime Types Allowed'}
            value={formik.values.media_types}
            onChange={formik.handleChange}
            //onBlur={formik.handleBlur}
            error={
                formik.touched.media_types && Boolean(formik.errors.media_types)
            }
            helperText={formik.touched.media_types && formik.errors.media_types}
        />
    );
};
