import React, { useCallback, useMemo } from 'react';
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
import { DatePicker } from '@mui/x-date-pickers';
import { Formik, useFormikContext } from 'formik';
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
    date_start: dateDefinedValidator('date_start'),
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

        const onSubmit = useCallback(
            async (values: ContestEditorFormValues) => {
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
                    modal.resolve(contest);
                    await modal.hide();
                } catch (e) {
                    logErr(e);
                    sendFlash('error', 'Error saving contest');
                }
            },
            [modal, sendFlash]
        );

        const formId = 'contestEditorForm';

        return (
            <Formik
                onSubmit={onSubmit}
                id={formId}
                validateOnBlur={false}
                validateOnChange={true}
                validationSchema={validationSchema}
                enableReinitialize={true}
                initialValues={{
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
                }}
            >
                <Dialog fullWidth {...muiDialogV5(modal)}>
                    <DialogTitle
                        component={Heading}
                        iconLeft={
                            loading ? <LoadingSpinner /> : <EmojiEventsIcon />
                        }
                    >
                        {`${
                            contest?.contest_id == EmptyUUID ? 'Create' : 'Edit'
                        } A Contest`}
                    </DialogTitle>

                    <DialogContent>
                        {loading ? (
                            <LoadingSpinner />
                        ) : (
                            <Stack spacing={2}>
                                <TitleField fullWidth isReadOnly={false} />
                                <DescriptionField
                                    fullWidth
                                    isReadOnly={false}
                                />
                                <Stack direction={'row'} spacing={2}>
                                    <PublicField fullWidth isReadOnly={false} />
                                    <HideSubmissionsField
                                        fullWidth
                                        isReadOnly={false}
                                    />
                                    <MaxSubmissionsField
                                        fullWidth
                                        isReadOnly={false}
                                    />
                                    <MinPermissionLevelField
                                        fullWidth
                                        isReadOnly={false}
                                    />
                                </Stack>
                                <Stack direction={'row'} spacing={2}>
                                    <VotingField fullWidth isReadOnly={false} />

                                    <DownVotesField fullWidth />
                                </Stack>

                                <Stack direction={'row'} spacing={2}>
                                    <DateStartField
                                        fullWidth
                                        isReadOnly={false}
                                    />
                                    <DateEndField
                                        fullWidth
                                        isReadOnly={false}
                                    />
                                </Stack>

                                <MimeTypeField fullWidth isReadOnly={false} />
                            </Stack>
                        )}
                    </DialogContent>
                    <DialogActions>
                        <CancelButton />
                        <ResetButton />
                        <SaveButton />
                    </DialogActions>
                </Dialog>
            </Formik>
        );
    }
);

const MaxSubmissionsField = ({ isReadOnly }: BaseFormikInputProps) => {
    const { values } = useFormikContext<ContestEditorFormValues>();
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
                value={values.max_submissions}
                onChange={(event) => {
                    console.log(event.target.value);
                    values.max_submissions = event.target.value as number;
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

const MinPermissionLevelField = ({ isReadOnly }: BaseFormikInputProps) => {
    const { values, handleChange } =
        useFormikContext<ContestEditorFormValues>();
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
                value={values.min_permission_level}
                onChange={handleChange}
            >
                <MenuItem value={PermissionLevel.User}>Logged In User</MenuItem>
                <MenuItem value={PermissionLevel.Editor}>Editor</MenuItem>
                <MenuItem value={PermissionLevel.Moderator}>Moderator</MenuItem>
                <MenuItem value={PermissionLevel.Admin}>Admin</MenuItem>
            </Select>
        </FormControl>
    );
};

const DateEndField = ({ isReadOnly }: BaseFormikInputProps) => {
    const { errors, touched, values, setFieldValue } =
        useFormikContext<ContestEditorFormValues>();
    return (
        <DatePicker
            disabled={isReadOnly ?? false}
            label="Date End"
            format="DD/MM/YYYY"
            value={values.date_end}
            //onChange={formik.handleChange}
            formatDensity={'dense'}
            //onError={(newError) => setError(newError)}
            onChange={async (value) => {
                await setFieldValue('date_end', value);
            }}
            slotProps={{
                textField: {
                    variant: 'outlined',
                    error: touched.date_end && Boolean(errors.date_end)
                }
            }}
        />
    );
};

const DateStartField = ({ isReadOnly }: BaseFormikInputProps) => {
    const { errors, touched, values, handleChange } =
        useFormikContext<ContestEditorFormValues>();
    return (
        <DatePicker
            disabled={isReadOnly ?? false}
            label="Date Start"
            format="DD/MM/YYYY"
            value={values.date_start}
            onChange={handleChange}
            //onError={(newError) => setError(newError)}
            //onChange={(value) => formik.setFieldValue("date_end", value, true)}
            slotProps={{
                textField: {
                    variant: 'outlined',
                    error: touched.date_start && Boolean(errors.date_start)
                    //helperText: formik.touched.date_end && formik.errors.date_end
                }
            }}
        />
    );
};

const PublicField = ({ isReadOnly }: BaseFormikInputProps) => {
    const { values, handleChange } =
        useFormikContext<ContestEditorFormValues>();
    return (
        <FormGroup>
            <FormControlLabel
                control={
                    <Checkbox
                        checked={values.public}
                        disabled={isReadOnly ?? false}
                    />
                }
                label="Public"
                name={'public'}
                onChange={handleChange}
            />
        </FormGroup>
    );
};

const HideSubmissionsField = ({ isReadOnly }: BaseFormikInputProps) => {
    const { values, handleChange } =
        useFormikContext<ContestEditorFormValues>();
    return (
        <FormGroup>
            <FormControlLabel
                control={
                    <Checkbox
                        disabled={isReadOnly ?? false}
                        checked={values.hide_submissions}
                    />
                }
                label="Hide Submissions"
                name={'hide_submissions'}
                onChange={handleChange}
            />
        </FormGroup>
    );
};

const VotingField = ({ isReadOnly }: BaseFormikInputProps) => {
    const { values, handleChange } =
        useFormikContext<ContestEditorFormValues>();
    return (
        <FormGroup>
            <FormControlLabel
                control={
                    <Checkbox
                        disabled={isReadOnly ?? false}
                        checked={values.voting}
                    />
                }
                label="Voting Allowed"
                name={'voting'}
                onChange={handleChange}
            />
        </FormGroup>
    );
};

const DownVotesField = ({ isReadOnly }: BaseFormikInputProps) => {
    const { values, handleChange } =
        useFormikContext<ContestEditorFormValues>();
    return (
        <FormGroup>
            <FormControlLabel
                control={
                    <Checkbox
                        disabled={isReadOnly ?? false}
                        checked={values.down_votes}
                    />
                }
                label="Down Votes Allowed"
                name={'down_votes'}
                onChange={handleChange}
            />
        </FormGroup>
    );
};

const TitleField = ({ isReadOnly }: BaseFormikInputProps) => {
    const { errors, touched, values, handleBlur, handleChange } =
        useFormikContext<ContestEditorFormValues>();
    return (
        <TextField
            fullWidth
            disabled={isReadOnly ?? false}
            name={'title'}
            id={'title'}
            label={'Title'}
            value={values.title}
            onChange={handleChange}
            onBlur={handleBlur}
            error={touched.title && Boolean(errors.title)}
            helperText={touched.title && errors.title}
        />
    );
};

const DescriptionField = ({ isReadOnly }: BaseFormikInputProps) => {
    const { errors, touched, values, handleBlur, handleChange } =
        useFormikContext<ContestEditorFormValues>();
    return (
        <TextField
            fullWidth
            multiline
            minRows={10}
            disabled={isReadOnly ?? false}
            name={'description'}
            id={'description'}
            label={'Description'}
            value={values.description}
            onChange={handleChange}
            onBlur={handleBlur}
            error={touched.description && Boolean(errors.description)}
            helperText={touched.description && errors.description}
        />
    );
};

const MimeTypeField = ({ isReadOnly }: BaseFormikInputProps) => {
    const { errors, touched, values, handleBlur, handleChange } =
        useFormikContext<ContestEditorFormValues>();
    return (
        <TextField
            fullWidth
            disabled={isReadOnly ?? false}
            name={'media_types'}
            id={'media_types'}
            label={'Mime Types Allowed'}
            value={values.media_types}
            onChange={handleChange}
            onBlur={handleBlur}
            error={touched.media_types && Boolean(errors.media_types)}
            helperText={touched.media_types && errors.media_types}
        />
    );
};
