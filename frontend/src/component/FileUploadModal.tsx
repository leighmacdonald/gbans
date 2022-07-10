import Modal from '@mui/material/Modal';
import React, { useCallback, useState } from 'react';
import Box from '@mui/material/Box';
import Typography from '@mui/material/Typography';
import TextField from '@mui/material/TextField';
import Button from '@mui/material/Button';
import Stack from '@mui/material/Stack';
import { fromByteArray } from 'base64-js';
import { UserUploadedFile } from '../api';
import { Nullable } from '../util/types';
import { logErr } from '../util/errors';

const style = {
    position: 'absolute' as const,
    top: '50%',
    left: '50%',
    transform: 'translate(-50%, -50%)',
    width: 400,
    bgcolor: 'background.paper',
    border: '2px solid #000',
    boxShadow: 24,
    p: 4
};

export interface FileUploadModalProps {
    open: boolean;
    setOpen: (isOpen: boolean) => void;
    onSave: (upload: UserUploadedFile) => void;
}

export const FileUploadModal = ({
    open,
    setOpen,
    onSave
}: FileUploadModalProps): JSX.Element => {
    const [url, setUrl] = useState('');
    const [upload, setUpload] = useState<Nullable<UserUploadedFile>>();
    const handleUploadedFile = useCallback(({ target }: any) => {
        const file = target.files[0];
        const reader = new FileReader();
        reader.addEventListener('load', (event) => {
            if (event?.target?.result) {
                setUpload({
                    content: fromByteArray(
                        new Uint8Array(event.target.result as ArrayBuffer)
                    ),
                    mime: file.type,
                    name: file.name,
                    size: file.size
                });
            }
        });

        reader.readAsArrayBuffer(file);
    }, []);

    const handleInsertButton = useCallback(() => {
        if (!upload && !url) {
            return;
        }
        if (url != '') {
            fetch(url)
                .then((resp) => {
                    console.log(resp);
                    resp.blob().then((blob) => {
                        blob.arrayBuffer().then((arrBuff) => {
                            onSave({
                                name: url.split('/').pop() || url,
                                mime: blob.type,
                                content: fromByteArray(new Uint8Array(arrBuff)),
                                size: blob.size
                            });
                        });
                    });
                })
                .catch(logErr);
        } else if (upload) {
            onSave(upload);
        }
    }, [onSave, upload, url]);

    const handleClose = () => setOpen(false);

    return (
        <Modal
            open={open}
            onClose={handleClose}
            aria-labelledby="modal-modal-title"
            aria-describedby="modal-modal-description"
        >
            <Box sx={style}>
                <Typography id="modal-modal-title" variant="h6" component="h2">
                    Upload An Image
                </Typography>
                <Typography id="modal-modal-description" sx={{ mt: 2 }}>
                    You can upload via pasting remote url, uploading a file or
                    pasting a file.
                </Typography>
                <Stack spacing={3}>
                    <label htmlFor="contained-button-file">
                        <input
                            id="contained-button-file"
                            accept="image/*"
                            type="file"
                            hidden={true}
                            onChange={handleUploadedFile}
                        />
                        <Button variant="contained" component="span">
                            Upload
                        </Button>
                    </label>
                    <Typography variant={'subtitle1'}>Remote URL</Typography>
                    <TextField
                        id="remote-file"
                        label="https://example.com/cat.jpg"
                        variant="outlined"
                        fullWidth
                        onChange={(event) => {
                            setUrl(event.target.value);
                        }}
                    />
                    <Button
                        variant={'contained'}
                        color={'primary'}
                        onClick={handleInsertButton}
                    >
                        Insert Image
                    </Button>
                </Stack>
            </Box>
        </Modal>
    );
};
