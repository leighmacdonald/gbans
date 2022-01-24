import React, { useCallback, useState } from 'react';
import TextField from '@mui/material/TextField';
import Button from '@mui/material/Button';
import DeleteOutlineIcon from '@mui/icons-material/DeleteOutline';
import Stack from '@mui/material/Stack';
import ListItemButton from '@mui/material/ListItemButton';
import ListItem from '@mui/material/ListItem';
import List from '@mui/material/List';
import Fab from '@mui/material/Fab';
import InputLabel from '@mui/material/InputLabel';
import ListItemText from '@mui/material/ListItemText';
import Select from '@mui/material/Select';
import FileUploadIcon from '@mui/icons-material/FileUpload';
import prettyBytes from 'pretty-bytes';
import { fromByteArray } from 'base64-js';
import Box from '@mui/material/Box';
import SendIcon from '@mui/icons-material/Send';
import FormControl from '@mui/material/FormControl';
import MenuItem from '@mui/material/MenuItem';
import { apiCreateReport, BanReason, BanReasons, UploadedFile } from '../api';

interface FormProps {
    uploadedFiles: UploadedFile[]; //(fileName:Blob) => Promise<void>, // callback taking a string and then dispatching a store actions
    setUploadedFiles: (files: UploadedFile[]) => void;
}

const FileUploaderForm: React.FunctionComponent<FormProps> = ({
    uploadedFiles,
    setUploadedFiles
}) => {
    const [selectedFiles, setSelectedFiles] = React.useState<File[]>([]);

    const handleUploadedFile = useCallback(
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
                }
            });

            reader.readAsArrayBuffer(target.files[0]);
            setSelectedFiles([...selectedFiles, target.files[0]]);
        },
        [selectedFiles, uploadedFiles, setUploadedFiles]
    );

    return (
        <Stack spacing={3}>
            <input
                accept=".png,image/jpeg,.webp,.dem,.stv"
                style={{
                    display: 'none'
                }}
                id="fileInput"
                type="file"
                onChange={handleUploadedFile}
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
    const [steamId, setSteamId] = useState<string>('');
    const [title, setTitle] = useState<string>('');
    const [reason, setReason] = useState<BanReason>(BanReason.Cheating);
    const [description, setDescription] = useState<string>('');
    const [uploadedFiles, setUploadedFiles] = useState<UploadedFile[]>([]);

    const submit = useCallback(async () => {
        await apiCreateReport({
            steam_id: steamId,
            title: title,
            description: description,
            media: uploadedFiles
        });
    }, [steamId, title, description, uploadedFiles]);

    return (
        <>
            <FormControl fullWidth>
                <TextField
                    id="title"
                    label={'Title'}
                    variant={'filled'}
                    margin={'normal'}
                    value={title}
                    onChange={(v) => {
                        setTitle(v.target.value);
                    }}
                />
            </FormControl>

            <FormControl fullWidth>
                <TextField
                    label="Steam Profile / Steam ID"
                    id="report_subject"
                    margin={'normal'}
                    error
                    helperText={'Invalid ID'}
                    variant={'filled'}
                    value={steamId}
                    onChange={(v) => {
                        setSteamId(v.target.value);
                    }}
                />
            </FormControl>

            <FormControl fullWidth margin={'normal'} variant={'filled'}>
                <InputLabel id="select_ban_reason_label">Ban Reason</InputLabel>
                <Select
                    labelId="select_ban_reason_label"
                    id="select_ban_reason"
                    value={reason}
                    variant={'filled'}
                    label={'Ban Reason'}
                    onChange={(v) => {
                        setReason(v.target.value as BanReason);
                    }}
                >
                    {[
                        BanReason.Custom,
                        BanReason.External,
                        BanReason.Cheating,
                        BanReason.Racism,
                        BanReason.Harassment,
                        BanReason.Exploiting,
                        BanReason.WarningsExceeded,
                        BanReason.Spam,
                        BanReason.Language
                    ].map((v) => {
                        return (
                            <MenuItem value={v} key={v}>
                                {BanReasons[v]}
                            </MenuItem>
                        );
                    })}
                </Select>
            </FormControl>
            <TextField
                label="Description"
                id="report_description"
                minRows={20}
                variant={'filled'}
                margin={'normal'}
                multiline
                fullWidth
                value={description}
                onChange={(v) => {
                    setDescription(v.target.value);
                }}
            />
            <FileUploaderForm
                setUploadedFiles={setUploadedFiles}
                uploadedFiles={uploadedFiles}
            />
            <Button
                fullWidth
                variant={'contained'}
                color={'primary'}
                onClick={submit}
                endIcon={<SendIcon />}
            >
                Submit Report
            </Button>
        </>
    );
};
