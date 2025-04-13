import { ChangeEvent, useCallback, useEffect, useState } from 'react';
import NiceModal, { muiDialogV5, useModal } from '@ebay/nice-modal-react';
import CloudUploadIcon from '@mui/icons-material/CloudUpload';
import EmojiEventsIcon from '@mui/icons-material/EmojiEvents';
import { Dialog, DialogActions, DialogContent, DialogTitle } from '@mui/material';
import Box from '@mui/material/Box';
import Button from '@mui/material/Button';
import CircularProgress from '@mui/material/CircularProgress';
import Grid from '@mui/material/Grid';
import Typography from '@mui/material/Typography';
import { useQuery } from '@tanstack/react-query';
import { apiContest } from '../../api';
import { apiSaveContestEntryMedia, UserUploadedFile } from '../../api/media';
import { AppError } from '../../error.tsx';
import { logErr } from '../../util/errors';
import { Nullable } from '../../util/types';
import { Heading } from '../Heading';
import { LinearProgressWithLabel } from '../LinearProgresWithLabel';
import { LoadingSpinner } from '../LoadingSpinner';

// interface ContestEntryFormValues {
//     contest_id: string;
//     body_md: string;
// }
//
// const validationSchema = yup.object({
//     body_md: minStringValidator('Description', 1)
// });

export const ContestEntryModal = NiceModal.create(({ contest_id }: { contest_id: number }) => {
    const [userUpload, setUserUpload] = useState<Nullable<UserUploadedFile>>();
    const [submittedOnce, setSubmittedOnce] = useState(false);
    const [progress, setProgress] = useState(0);
    const [progressTotal, setProgressTotal] = useState(100);
    const [uploadInProgress, setUploadInProgress] = useState(false);
    const [name, setName] = useState('');
    const [assetID, setAssetID] = useState('');
    const [assetError, setAssetError] = useState('');

    const { data: contest, isLoading } = useQuery({
        queryKey: ['contest', { contest_id }],
        queryFn: async () => {
            return await apiContest(contest_id);
        }
    });

    const modal = useModal();
    // const { sendFlash } = useUserFlashCtx();

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
                    // TODO use new Blob() to upload via form.
                    // setUserUpload({
                    //     content: fromByteArray(new Uint8Array(event.target.result as ArrayBuffer)),
                    //     mime: file.type,
                    //     name: file.name,
                    //     size: file.size
                    // });
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
        const abortController = new AbortController();
        const uploadMedia = async () => {
            try {
                const media = await apiSaveContestEntryMedia(contest_id, userUpload);
                setAssetID(media.asset_id);
                setAssetError('');
            } catch (err) {
                if (err instanceof AppError) {
                    setAssetError(err.message);
                } else {
                    logErr(err);
                }
                setName('');
                setSubmittedOnce(false);
                setUserUpload(undefined);
            }
        };

        uploadMedia().catch(logErr);

        return () => abortController.abort();
    }, [contest_id, userUpload]);

    // const onSubmit = useCallback(
    //     async (values: ContestEntryFormValues) => {
    //         setSubmittedOnce(true);
    //         if (assetID == '') {
    //             return;
    //         }
    //
    //         try {
    //             const contest = await apiContestEntrySave(values.contest_id, values.body_md, assetID);
    //             sendFlash('success', `Entry created successfully (${contest.contest_id}`);
    //             await modal.hide();
    //         } catch (err) {
    //             if (err instanceof AppError) {
    //                 sendFlash('error', err.message);
    //             } else {
    //                 logErr(err);
    //             }
    //             await modal.hide();
    //         }
    //     },
    //     [assetID, modal, sendFlash]
    // );

    // const formId = 'contestSubmitForm';

    return (
        // <Formik
        //     onSubmit={onSubmit}
        //     id={formId}
        //     initialValues={{
        //         contest_id: contest?.contest_id ?? EmptyUUID,
        //         body_md: ''
        //     }}
        //     validateOnBlur={false}
        //     validateOnChange={false}
        //     validationSchema={validationSchema}
        //     enableReinitialize={true}
        // >
        <Dialog fullWidth {...muiDialogV5(modal)}>
            <DialogTitle component={Heading} iconLeft={isLoading ? <LoadingSpinner /> : <EmojiEventsIcon />}>
                {`Submit Entry For: ${contest?.title}`}
            </DialogTitle>

            <DialogContent>
                {isLoading ? (
                    <LoadingSpinner />
                ) : (
                    <Grid container spacing={2}>
                        <Grid size={{ xs: 12 }}>{/*<MDBodyField fileUpload={false} />*/}</Grid>
                        <Grid size={{ xs: 4 }}>
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
                                    disabled={Boolean(userUpload) || uploadInProgress}
                                    startIcon={
                                        uploadInProgress ? (
                                            <CircularProgress
                                                size={'20px'}
                                                variant={'determinate'}
                                                color={'secondary'}
                                                value={(progress / progressTotal) * 100}
                                            />
                                        ) : (
                                            <CloudUploadIcon />
                                        )
                                    }
                                >
                                    {uploadInProgress ? 'Uploading...' : 'Select File'}
                                </Button>
                            </label>
                        </Grid>
                        <Grid size={{ xs: 8 }}>
                            {uploadInProgress ? (
                                <LinearProgressWithLabel value={(progress / progressTotal) * 100} />
                            ) : (
                                <Box display="flex" alignItems="center">
                                    <Typography variant={'button'}>{userUpload?.name}</Typography>
                                </Box>
                            )}
                        </Grid>
                        {assetError != '' && (
                            <Grid size={{ xs: 12 }}>
                                <Typography variant={'body2'} color={'error'}>
                                    {assetError}
                                </Typography>
                            </Grid>
                        )}
                        {submittedOnce && assetID == '' && (
                            <Grid size={{ xs: 12 }}>
                                <Box display="flex" alignItems="center">
                                    <Typography variant={'body1'} color={'error'} fontSize={'smaller'}>
                                        Must upload file
                                    </Typography>
                                </Box>
                            </Grid>
                        )}
                    </Grid>
                )}
            </DialogContent>
            <DialogActions>
                {/*<CancelButton />*/}
                {/*<ResetButton />*/}
                {/*<SubmitButton disabled={uploadInProgress} />*/}
            </DialogActions>
        </Dialog>
        // </Formik>
    );
});
