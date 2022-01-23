import React, { useCallback } from 'react';
import TextField from '@mui/material/TextField';
import Button from '@mui/material/Button';
import DeleteOutlineIcon from '@mui/icons-material/DeleteOutline';
import Stack from '@mui/material/Stack';
import ListItemButton from '@mui/material/ListItemButton';
import ListItem from '@mui/material/ListItem';
import List from '@mui/material/List';
import { Fab, ListItemText } from '@mui/material';
import FileUploadIcon from '@mui/icons-material/FileUpload';
import prettyBytes from 'pretty-bytes';
import { fromByteArray } from 'base64-js';
import Box from '@mui/material/Box';
import SendIcon from '@mui/icons-material/Send';
interface FormProps {
    saveFace: any; //(fileName:Blob) => Promise<void>, // callback taking a string and then dispatching a store actions
}

interface UploadedFile {
    content: string;
    name: string;
    mime: string;
    size: number;
}

const FileField: React.FunctionComponent<FormProps> = () => {
    const [selectedFiles, setSelectedFiles] = React.useState<File[]>([]);
    const [uploadedFiles, setUploadedFiles] = React.useState<UploadedFile[]>(
        []
    );

    const handleCapture = useCallback(
        ({ target }: any) => {
            const f = target.files[0];
            const reader = new FileReader();

            reader.addEventListener('load', function (e) {
                if (e?.target?.result) {
                    const bytes = fromByteArray(
                        new Uint8Array(e.target.result as ArrayBuffer)
                    );
                    const x = [
                        ...uploadedFiles,
                        {
                            content: bytes,
                            mime: f.type,
                            name: f.name,
                            size: f.size
                        }
                    ];
                    setUploadedFiles(x);
                    console.log(`${f.name} ${prettyBytes(f.size)} ${f.mime}`);
                    console.log(`${bytes}`);
                }
            });

            reader.readAsArrayBuffer(target.files[0]);
            setSelectedFiles([...selectedFiles, target.files[0]]);
        },
        [selectedFiles, uploadedFiles]
    );

    // const handleSubmit = () => {
    //     saveFace(selectedFiles);
    // };

    return (
        <Stack spacing={3}>
            <input
                accept="image/png,image/jpeg,image/webp,.dem,.stv"
                style={{
                    display: 'none'
                }}
                id="fileInput"
                type="file"
                onChange={handleCapture}
            />

            <Box sx={{ '& > :not(style)': { m: 1 } }}>
                <label htmlFor="fileInput">
                    <Fab
                        variant={'extended'}
                        size="small"
                        color={'secondary'}
                        aria-label="upload"
                        onClick={() => {
                            const input = document.getElementById('fileInput');
                            input?.dispatchEvent(
                                new MouseEvent('click', {
                                    bubbles: true,
                                    cancelable: false,
                                    view: window
                                })
                            );
                        }}
                    >
                        <FileUploadIcon sx={{ mr: 1 }} />
                        Upload Evidence
                    </Fab>
                </label>
            </Box>
            <List>
                {selectedFiles.map((f, idx) => {
                    return (
                        <ListItem key={f.name}>
                            <ListItemButton
                                onClick={() => {
                                    setSelectedFiles(
                                        selectedFiles.filter((_, i) => {
                                            return i !== idx;
                                        })
                                    );
                                }}
                            >
                                <DeleteOutlineIcon />
                            </ListItemButton>
                            <ListItemText>{f.name}</ListItemText>
                            <ListItemText>{prettyBytes(f.size)}</ListItemText>
                        </ListItem>
                    );
                })}
            </List>
        </Stack>
    );
};

export const ReportForm = (): JSX.Element => {
    return (
        <>
            <TextField
                fullWidth
                label="Steam Profile / Steam ID"
                id="report_subject"
                variant={'filled'}
            />
            <TextField
                label="Description"
                id="report_description"
                minRows={20}
                variant={'filled'}
                multiline
                fullWidth
            />
            <FileField
                saveFace={() => {
                    alert('save');
                }}
            />
            <Button
                fullWidth
                variant={'contained'}
                color={'primary'}
                endIcon={<SendIcon />}
            >
                Submit Report
            </Button>
        </>
    );
};
