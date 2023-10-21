import React, { ChangeEvent, useCallback, useEffect, useState } from 'react';
import NiceModal, { useModal, muiDialogV5 } from '@ebay/nice-modal-react';
import CloudUploadIcon from '@mui/icons-material/CloudUpload';
import EmojiEventsIcon from '@mui/icons-material/EmojiEvents';
import {
    Dialog,
    DialogActions,
    DialogContent,
    DialogTitle
} from '@mui/material';
import Box from '@mui/material/Box';
import Button from '@mui/material/Button';
import CircularProgress from '@mui/material/CircularProgress';
import TextField from '@mui/material/TextField';
import Typography from '@mui/material/Typography';
import Grid from '@mui/material/Unstable_Grid2';
import { fromByteArray } from 'base64-js';
import { useFormik } from 'formik';
import * as yup from 'yup';
import { apiContestEntrySave, EmptyUUID, useContest } from '../../api';
import { apiSaveContestEntryMedia, UserUploadedFile } from '../../api/media';
import { useUserFlashCtx } from '../../contexts/UserFlashCtx';
import { logErr } from '../../util/errors';
import { Nullable } from '../../util/types';
import { Heading } from '../Heading';
import { LinearProgressWithLabel } from '../LinearProgresWithLabel';
import { LoadingSpinner } from '../LoadingSpinner';
import { BaseFormikInputProps } from '../formik/SteamIdField';
import { minStringValidator } from '../formik/Validator';
import { CancelButton, ResetButton, SaveButton } from './Buttons';

interface ContestEntryFormValues {
    contest_id: string;
    description: string;
}

const validationSchema = yup.object({
    description: minStringValidator('Description', 1)
});

export const ContestEntryModal = NiceModal.create(
    ({ contest_id }: { contest_id: string }) => {
        const [userUpload, setUserUpload] =
            useState<Nullable<UserUploadedFile>>();
        const [submittedOnce, setSubmittedOnce] = useState(false);
        const [progress, setProgress] = useState(0);
        const [progressTotal, setProgressTotal] = useState(100);
        const [uploadInProgress, setUploadInProgress] = useState(false);
        const [name, setName] = useState('');
        const [assetID, setAssetID] = useState('');
        const { loading, contest } = useContest(contest_id);
        const modal = useModal();
        const { sendFlash } = useUserFlashCtx();

        const handleUploadedFile = useCallback(
            ({ target }: ChangeEvent<HTMLInputElement>) => {
                if (!target.files) {
                    return;
                }
                setUploadInProgress(true);
                const file = target.files[0];
                const reader = new FileReader();
                reader.onprogress = (ev) => {
                    setProgress(ev.loaded);
                    setProgressTotal(ev.total);
                };
                reader.addEventListener('load', (event) => {
                    if (event?.target?.result) {
                        if (!name) {
                            setName(file.name);
                        }
                        setUserUpload({
                            content: fromByteArray(
                                new Uint8Array(
                                    event.target.result as ArrayBuffer
                                )
                            ),
                            mime: file.type,
                            name: file.name,
                            size: file.size
                        });
                        setProgress(progressTotal);
                    }

                    setUploadInProgress(false);
                });

                reader.readAsArrayBuffer(file);
            },
            [progressTotal, name]
        );

        useEffect(() => {
            if (!userUpload) {
                return;
            }
            apiSaveContestEntryMedia(contest_id, userUpload)
                .then((media) => {
                    setAssetID(media.asset.asset_id);
                })
                .catch((e) => {
                    logErr(e);
                });
        }, [contest_id, userUpload]);

        const formik = useFormik<ContestEntryFormValues>({
            initialValues: {
                contest_id: contest?.contest_id ?? EmptyUUID,
                description: contest?.description ?? ''
            },
            validateOnBlur: false,
            validateOnChange: false,
            validationSchema: validationSchema,
            enableReinitialize: true,
            onSubmit: async (values) => {
                setSubmittedOnce(true);
                if (assetID == '') {
                    return;
                }

                try {
                    const contest = await apiContestEntrySave(
                        values.contest_id != '' ? values.contest_id : EmptyUUID,
                        values.description,
                        assetID
                    );
                    sendFlash(
                        'success',
                        `Entry created successfully (${contest.contest_id}`
                    );
                    await modal.hide();
                } catch (e) {
                    logErr(e);
                    sendFlash('error', 'Error saving entry');
                }
            }
        });

        const formId = 'contestSubmitForm';

        return (
            <form onSubmit={formik.handleSubmit} id={formId}>
                <Dialog fullWidth {...muiDialogV5(modal)}>
                    <DialogTitle
                        component={Heading}
                        iconLeft={
                            loading ? <LoadingSpinner /> : <EmojiEventsIcon />
                        }
                    >
                        {`Submit Entry For: ${contest?.title}`}
                    </DialogTitle>

                    <DialogContent>
                        {loading ? (
                            <LoadingSpinner />
                        ) : (
                            <Grid container spacing={2}>
                                <Grid xs={12}>
                                    <DescriptionField
                                        formik={formik}
                                        fullWidth
                                        isReadOnly={false}
                                    />
                                </Grid>
                                <Grid xs={4}>
                                    <label htmlFor="contained-button-file">
                                        <input
                                            id="contained-button-file"
                                            accept="image/*,video/*"
                                            type="file"
                                            hidden={true}
                                            onChange={handleUploadedFile}
                                        />
                                        <Button
                                            variant="contained"
                                            component="span"
                                            fullWidth
                                            disabled={
                                                Boolean(userUpload) ||
                                                uploadInProgress
                                            }
                                            startIcon={
                                                uploadInProgress ? (
                                                    <CircularProgress
                                                        size={'20px'}
                                                        variant={'determinate'}
                                                        color={'secondary'}
                                                        value={
                                                            (progress /
                                                                progressTotal) *
                                                            100
                                                        }
                                                    />
                                                ) : (
                                                    <CloudUploadIcon />
                                                )
                                            }
                                        >
                                            {uploadInProgress
                                                ? 'Uploading...'
                                                : 'Select File'}
                                        </Button>
                                    </label>
                                </Grid>
                                <Grid xs={8}>
                                    {uploadInProgress ? (
                                        <LinearProgressWithLabel
                                            value={
                                                (progress / progressTotal) * 100
                                            }
                                        />
                                    ) : (
                                        <Box display="flex" alignItems="center">
                                            <Typography variant={'button'}>
                                                {userUpload?.name}
                                            </Typography>
                                        </Box>
                                    )}
                                </Grid>
                                {submittedOnce && assetID == '' && (
                                    <Grid xs={12}>
                                        <Box display="flex" alignItems="center">
                                            <Typography
                                                variant={'body1'}
                                                color={'error'}
                                                fontSize={'smaller'}
                                            >
                                                Must upload file
                                            </Typography>
                                        </Box>
                                    </Grid>
                                )}
                            </Grid>
                        )}
                    </DialogContent>
                    <DialogActions>
                        <CancelButton onClick={modal.hide} />
                        <ResetButton onClick={formik.resetForm} />
                        <SaveButton
                            onClick={formik.submitForm}
                            disabled={uploadInProgress}
                        />
                    </DialogActions>
                </Dialog>
            </form>
        );
    }
);

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
