import { useCallback, useMemo } from 'react';
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
import { Formik, useFormikContext } from 'formik';
import * as yup from 'yup';
import { EmptyUUID, PermissionLevel } from '../../api';
import { apiContestSave } from '../../api';
import { useContest } from '../../hooks/useContest.ts';
import { useUserFlashCtx } from '../../hooks/useUserFlashCtx.ts';
import { logErr } from '../../util/errors';
import { Heading } from '../Heading';
import { LoadingSpinner } from '../LoadingSpinner';
import { DateEndField } from '../formik/DateEndField';
import { DateStartField } from '../formik/DateStartField';
import { DescriptionField } from '../formik/DescriptionField';
import { BaseFormikInputProps } from '../formik/SourceIDField.tsx';
import { TitleField } from '../formik/TitleField';
import {
    boolDefinedValidator,
    dateAfterValidator,
    dateDefinedValidator,
    mimeTypesValidator,
    minStringValidator,
    numberValidator,
    permissionValidator
} from '../formik/Validator';
import { CancelButton, ResetButton, SubmitButton } from './Buttons';

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
                        num_entries: 0,
                        updated_on: new Date(),
                        created_on: new Date()
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
                        contest?.min_permission_level != undefined
                            ? contest?.min_permission_level
                            : PermissionLevel.User
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
                                <TitleField />
                                <DescriptionField />
                                <Stack direction={'row'} spacing={2}>
                                    <PublicField />
                                    <HideSubmissionsField />
                                    <MaxSubmissionsField />
                                    <MinPermissionLevelField />
                                </Stack>
                                <Stack direction={'row'} spacing={2}>
                                    <VotingField />
                                    <DownVotesField />
                                </Stack>

                                <Stack direction={'row'} spacing={2}>
                                    <DateStartField />
                                    <DateEndField />
                                </Stack>

                                <MimeTypeField />
                            </Stack>
                        )}
                    </DialogContent>
                    <DialogActions>
                        <CancelButton />
                        <ResetButton />
                        <SubmitButton />
                    </DialogActions>
                </Dialog>
            </Formik>
        );
    }
);

const MaxSubmissionsField = () => {
    const { handleChange, values, touched, errors } =
        useFormikContext<ContestEditorFormValues>();
    return (
        <FormControl fullWidth>
            <InputLabel id="max_submissions-label">
                Maximum Submissions Per User
            </InputLabel>
            <Select<number>
                name={`max_submissions`}
                labelId={`max_submissions-label`}
                id={`max_submissions`}
                label={'Maximum Submissions Per User'}
                error={
                    touched.max_submissions && Boolean(errors.max_submissions)
                }
                value={values.max_submissions}
                onChange={handleChange}
            >
                {[-1, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10].map((c) => (
                    <MenuItem key={`max-subs-${c}`} value={c}>
                        {c < 0 ? 'Unlimited' : c}
                    </MenuItem>
                ))}
            </Select>
            <FormHelperText>
                {touched.max_submissions &&
                    Boolean(errors.max_submissions) &&
                    errors.max_submissions}
            </FormHelperText>
        </FormControl>
    );
};

const MinPermissionLevelField = () => {
    const { values, touched, errors, handleChange } =
        useFormikContext<ContestEditorFormValues>();
    return (
        <FormControl fullWidth>
            <InputLabel id="min_permission_level-label">
                Minimum permissions required to submit
            </InputLabel>
            <Select<number>
                name={`min_permission_level`}
                labelId={`min_permission_level-label`}
                id={`min_permission_level`}
                label={'Minimum permissions required to submit'}
                value={values.min_permission_level}
                onChange={handleChange}
            >
                <MenuItem value={PermissionLevel.User}>Logged In User</MenuItem>
                <MenuItem value={PermissionLevel.Editor}>Editor</MenuItem>
                <MenuItem value={PermissionLevel.Moderator}>Moderator</MenuItem>
                <MenuItem value={PermissionLevel.Admin}>Admin</MenuItem>
            </Select>
            <FormHelperText>
                {touched.min_permission_level &&
                    Boolean(errors.min_permission_level) &&
                    errors.min_permission_level}
            </FormHelperText>
        </FormControl>
    );
};

const PublicField = () => {
    const { values, handleChange } =
        useFormikContext<ContestEditorFormValues>();
    return (
        <FormGroup>
            <FormControlLabel
                control={<Checkbox checked={values.public} />}
                label="Public"
                name={'public'}
                onChange={handleChange}
            />
        </FormGroup>
    );
};

const HideSubmissionsField = () => {
    const { values, handleChange } =
        useFormikContext<ContestEditorFormValues>();
    return (
        <FormGroup>
            <FormControlLabel
                control={<Checkbox checked={values.hide_submissions} />}
                label="Hide Submissions"
                name={'hide_submissions'}
                onChange={handleChange}
            />
        </FormGroup>
    );
};

const VotingField = () => {
    const { values, handleChange } =
        useFormikContext<ContestEditorFormValues>();
    return (
        <FormGroup>
            <FormControlLabel
                control={<Checkbox checked={values.voting} />}
                label="Voting Allowed"
                name={'voting'}
                onChange={handleChange}
            />
        </FormGroup>
    );
};

const DownVotesField = () => {
    const { values, handleChange } =
        useFormikContext<ContestEditorFormValues>();

    return (
        <FormGroup>
            <FormControlLabel
                disabled={!values.voting}
                control={<Checkbox checked={values.down_votes} />}
                label="Down Votes Allowed"
                name={'down_votes'}
                onChange={handleChange}
            />
        </FormGroup>
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

export default ContestEditor;
