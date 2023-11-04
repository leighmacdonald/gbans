import React, { ChangeEvent, useCallback, useState, JSX } from 'react';
import NiceModal, { muiDialogV5, useModal } from '@ebay/nice-modal-react';
import {
    Dialog,
    DialogActions,
    DialogContent,
    DialogTitle,
    Divider
} from '@mui/material';
import Button from '@mui/material/Button';
import LinearProgress from '@mui/material/LinearProgress';
import Paper from '@mui/material/Paper';
import Stack from '@mui/material/Stack';
import Typography from '@mui/material/Typography';
import { useTheme } from '@mui/material/styles';
import styled from '@mui/system/styled';
import { fromByteArray } from 'base64-js';
import { Formik } from 'formik';
import { UserUploadedFile } from '../../api/media';
import { useUserFlashCtx } from '../../contexts/UserFlashCtx';
import { Nullable } from '../../util/types';
import { Heading } from '../Heading';
import { NameField } from '../formik/NameField';
import { CancelButton, ResetButton, SaveButton } from './Buttons';

const MethodPaper = styled(Paper)(({ theme }) => ({
    padding: theme.spacing(1),
    textAlign: 'center',
    lineHeight: '60px',
    minWidth: '30%'
}));

interface FileUploadModalProps {
    name: string;
}

export const FileUploadModal = NiceModal.create((): JSX.Element => {
    const theme = useTheme();
    const { sendFlash } = useUserFlashCtx();
    const [upload, setUpload] = useState<Nullable<UserUploadedFile>>();
    const [progress, setProgress] = useState(0);
    const [progressTotal, setProgressTotal] = useState(100);
    const [uploadInProgress, setUploadInProgress] = useState(false);
    const [name, setName] = useState('');
    const [enabledPanel, setEnabledPanel] = useState<
        'all' | 'file' | 'url' | 'paste'
    >('all');
    const modal = useModal();
    const reset = async () => {
        setEnabledPanel('all');
        setName('');
        setProgress(0);
        setUploadInProgress(false);
        setUpload(null);
    };

    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    const handleUploadedFile = useCallback(
        ({ target }: ChangeEvent<HTMLInputElement>) => {
            if (!target.files) {
                return;
            }
            const file = target.files[0];
            const reader = new FileReader();
            reader.addEventListener('load', (event) => {
                if (event?.target?.result) {
                    if (!name) {
                        setName(file.name);
                    }
                    setUpload({
                        content: fromByteArray(
                            new Uint8Array(event.target.result as ArrayBuffer)
                        ),
                        mime: file.type,
                        name: file.name,
                        size: file.size
                    });
                    setEnabledPanel('file');
                }
            });

            reader.readAsArrayBuffer(file);
        },
        [name]
    );

    const handlePaste = useCallback((event: React.ClipboardEvent) => {
        setUploadInProgress(true);
        const items = event.clipboardData.items;
        // eslint-disable-next-line no-loops/no-loops
        for (const index in items) {
            const item = items[index];
            if (item.kind === 'file') {
                const blob = item.getAsFile();
                if (!blob) {
                    return;
                }
                const reader = new FileReader();
                reader.onprogress = (ev) => {
                    setProgress(ev.loaded);
                    setProgressTotal(ev.total);
                };

                reader.onload = (event: ProgressEvent<FileReader>) => {
                    if (event?.target?.result) {
                        setEnabledPanel('paste');
                        const content = fromByteArray(
                            new Uint8Array(event.target.result as ArrayBuffer)
                        );
                        setUpload({
                            content: content,
                            mime: '__unknown__',
                            name: '__unknown__',
                            size: content.length
                        });
                    }
                    setUploadInProgress(false);
                }; // data url!
                reader.readAsArrayBuffer(blob);
            }
        }
    }, []);

    const handleSave = useCallback(async () => {
        if (!upload) {
            sendFlash('error', 'Must select only 1 of the 3 upload options');
            return;
        }
        modal.resolve(upload);
        await modal.hide();
    }, [modal, sendFlash, upload]);

    const onSubmit = useCallback(async (values: FileUploadModalProps) => {
        console.log(values);
    }, []);

    return (
        <Formik<FileUploadModalProps>
            initialValues={{ name: '' }}
            onSubmit={onSubmit}
            onReset={async () => {
                await reset();
            }}
        >
            <Dialog
                aria-labelledby="modal-modal-title"
                aria-describedby="modal-modal-description"
                onPaste={handlePaste}
                fullWidth
                maxWidth={'lg'}
                {...muiDialogV5(modal)}
            >
                <DialogTitle component={Heading}>Upload An Image</DialogTitle>
                <DialogContent>
                    <Stack spacing={2}>
                        <Typography id="modal-modal-description" sx={{ mt: 2 }}>
                            You can upload evidence screenshots by choosing one
                            of the 3 methods below.
                        </Typography>
                        <Divider orientation="horizontal" flexItem />
                        <Stack
                            direction={{ xs: 'column', sm: 'row' }}
                            justifyContent="space-evenly"
                            alignItems="stretch"
                            divider={
                                <Divider orientation="vertical" flexItem />
                            }
                            spacing={2}
                        >
                            <MethodPaper elevation={1}>
                                <Typography variant={'subtitle1'}>
                                    File Upload
                                </Typography>
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
                                            !['all', 'file'].includes(
                                                enabledPanel
                                            )
                                        }
                                    >
                                        Select File
                                    </Button>
                                </label>
                            </MethodPaper>

                            <MethodPaper>
                                <Typography variant={'subtitle1'}>
                                    Paste
                                </Typography>
                                <Typography
                                    variant={'body2'}
                                    color={'disabled'}
                                    sx={{
                                        color: !['all', 'paste'].includes(
                                            enabledPanel
                                        )
                                            ? theme.palette.grey['500']
                                            : theme.typography.body2.color
                                    }}
                                >
                                    You can capture a screen shot (Windows
                                    screenshot shortcut: <kbd>win+shift+s</kbd>)
                                    and paste it anywhere in the window using{' '}
                                    <kbd>ctrl+v</kbd>.
                                </Typography>
                            </MethodPaper>
                        </Stack>

                        <NameField />

                        <Stack
                            spacing={2}
                            direction="row-reverse"
                            justifyContent="flex-start"
                            alignItems="flex-start"
                        >
                            {uploadInProgress && (
                                <LinearProgress
                                    value={progress}
                                    valueBuffer={progressTotal}
                                />
                            )}
                        </Stack>
                    </Stack>
                </DialogContent>
                <DialogActions>
                    <CancelButton />
                    <ResetButton />
                    <SaveButton onClick={handleSave} />
                </DialogActions>
            </Dialog>
        </Formik>
    );
});
