import React, { ChangeEvent, useCallback, useState, JSX } from 'react';
import Typography from '@mui/material/Typography';
import TextField from '@mui/material/TextField';
import Button from '@mui/material/Button';
import Stack from '@mui/material/Stack';
import { fromByteArray } from 'base64-js';
import { Nullable } from '../util/types';
import { logErr } from '../util/errors';
import { UserUploadedFile } from '../api/media';
import LinearProgress from '@mui/material/LinearProgress';
import Paper from '@mui/material/Paper';
import {
    Dialog,
    DialogActions,
    DialogContent,
    DialogTitle,
    Divider
} from '@mui/material';
import styled from '@mui/system/styled';
import { useUserFlashCtx } from '../contexts/UserFlashCtx';
import { Heading } from './Heading';
import useTheme from '@mui/material/styles/useTheme';

export interface FileUploadModalProps {
    open: boolean;
    setOpen: (isOpen: boolean) => void;
    onSave: (upload: UserUploadedFile, onSuccess: () => void) => void;
}

const MethodPaper = styled(Paper)(({ theme }) => ({
    padding: theme.spacing(1),
    textAlign: 'center',
    lineHeight: '60px',
    minWidth: '30%'
}));

export const FileUploadModal = ({
    open,
    setOpen,
    onSave
}: FileUploadModalProps): JSX.Element => {
    const theme = useTheme();
    const { sendFlash } = useUserFlashCtx();
    const [url, setUrl] = useState('');
    const [upload, setUpload] = useState<Nullable<UserUploadedFile>>();
    const [progress, setProgress] = useState(0);
    const [progressTotal, setProgressTotal] = useState(100);
    const [uploadInProgress, setUploadInProgress] = useState(false);
    const [name, setName] = useState('');
    const [enabledPanel, setEnabledPanel] = useState<
        'all' | 'file' | 'url' | 'paste'
    >('all');

    const reset = () => {
        setEnabledPanel('all');
        setUrl('');
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

    const handleDownloadButton = useCallback(() => {
        if (url) {
            setUploadInProgress(true);
            setEnabledPanel('url');
            fetch(url)
                .then((resp) => {
                    resp.blob().then((blob) => {
                        blob.arrayBuffer().then((arrBuff) => {
                            const u = url.split('/').pop() || url;
                            if (!name) {
                                setName(u);
                            }
                            setUpload({
                                name: u,
                                mime: blob.type,
                                content: fromByteArray(new Uint8Array(arrBuff)),
                                size: blob.size
                            });
                        });
                    });
                })
                .catch(logErr)
                .finally(() => {
                    setUploadInProgress(false);
                });
        }
    }, [name, url]);

    const handleSave = useCallback(() => {
        if (!upload) {
            sendFlash('error', 'Must select only 1 of the 3 upload options');
            return;
        }
        onSave(upload, reset);
    }, [onSave, sendFlash, upload]);

    const handleClose = () => setOpen(false);

    return (
        <Dialog
            open={open}
            onClose={handleClose}
            aria-labelledby="modal-modal-title"
            aria-describedby="modal-modal-description"
            onPaste={handlePaste}
            fullWidth
            maxWidth={'lg'}
        >
            <DialogTitle component={Heading}>Upload An Image</DialogTitle>
            <DialogContent>
                <Stack spacing={2}>
                    <Typography id="modal-modal-description" sx={{ mt: 2 }}>
                        You can upload evidence screenshots by choosing one of
                        the 3 methods below.
                    </Typography>
                    <Divider orientation="horizontal" flexItem />
                    <Stack
                        direction={{ xs: 'column', sm: 'row' }}
                        justifyContent="space-evenly"
                        alignItems="stretch"
                        divider={<Divider orientation="vertical" flexItem />}
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
                                        !['all', 'file'].includes(enabledPanel)
                                    }
                                >
                                    Select File
                                </Button>
                            </label>
                        </MethodPaper>

                        <MethodPaper elevation={1}>
                            <Typography variant={'subtitle1'}>
                                Remote URL
                            </Typography>
                            <Stack direction={'row'}>
                                <TextField
                                    id="remote-file"
                                    label="https://example.com/cat.jpg"
                                    variant="outlined"
                                    fullWidth
                                    value={url}
                                    disabled={
                                        !['all', 'url'].includes(enabledPanel)
                                    }
                                    onChange={(event) => {
                                        setEnabledPanel(
                                            event.target.value ? 'url' : 'all'
                                        );
                                        setUrl(event.target.value);
                                    }}
                                />
                                <Button
                                    onClick={handleDownloadButton}
                                    disabled={url.length == 0}
                                >
                                    Load
                                </Button>
                            </Stack>
                        </MethodPaper>

                        <MethodPaper>
                            <Typography variant={'subtitle1'}>Paste</Typography>
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
                                screenshot shortcut: <kbd>win+shift+s</kbd>) and
                                paste it anywhere in the window using{' '}
                                <kbd>ctrl+v</kbd>.
                            </Typography>
                        </MethodPaper>
                    </Stack>

                    <TextField
                        id="name"
                        label="Optional Name"
                        variant="outlined"
                        fullWidth
                        value={name}
                        onChange={(event) => {
                            setName(event.target.value);
                        }}
                    />

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
                <Button variant={'contained'} color={'warning'} onClick={reset}>
                    Reset
                </Button>

                <Button
                    variant={'contained'}
                    color={'error'}
                    onClick={() => {
                        reset();
                        setOpen(false);
                    }}
                >
                    Cancel
                </Button>

                <Button
                    variant={'contained'}
                    color={'success'}
                    onClick={handleSave}
                    disabled={!upload || !upload.content}
                >
                    Insert Image
                </Button>
            </DialogActions>
        </Dialog>
    );
};
