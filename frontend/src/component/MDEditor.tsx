import React, { useEffect, useState, useCallback } from 'react';
import NiceModal from '@ebay/nice-modal-react';
import FormatBoldIcon from '@mui/icons-material/FormatBold';
import FormatIndentDecreaseIcon from '@mui/icons-material/FormatIndentDecrease';
import FormatIndentIncreaseIcon from '@mui/icons-material/FormatIndentIncrease';
import FormatUnderlinedIcon from '@mui/icons-material/FormatUnderlined';
import ImageIcon from '@mui/icons-material/Image';
import Box from '@mui/material/Box';
import Button from '@mui/material/Button';
import ButtonGroup from '@mui/material/ButtonGroup';
import IconButton from '@mui/material/IconButton';
import Stack from '@mui/material/Stack';
import Tab from '@mui/material/Tab';
import Tabs from '@mui/material/Tabs';
import TextField from '@mui/material/TextField';
import Tooltip from '@mui/material/Tooltip';
import { apiSaveMedia, UserUploadedFile } from '../api/media';
import { renderMarkdown } from '../api/wiki';
import { useUserFlashCtx } from '../contexts/UserFlashCtx';
import { logErr } from '../util/errors';
import { TabPanel } from './TabPanel';
import { ModalFileUpload } from './modal';

interface MDEditorProps {
    initialBodyMDValue: string;
    // onSave is called when the save / accept button is hit. The onSuccess
    // function should be called if the save succeeded to clean up the message state
    // of the editor
    onSave: (body_md: string, onSuccess?: () => void) => void;
    cancelEnabled?: boolean;
    onCancel?: () => void;
    saveLabel?: string;
    cancelLabel?: string;
}

export const MDEditor = ({
    onSave,
    onCancel,
    cancelEnabled,
    initialBodyMDValue,
    saveLabel,
    cancelLabel
}: MDEditorProps) => {
    const [setTabValue, setTabSetTabValue] = useState(0);
    const [bodyHTML, setBodyHTML] = useState('');
    const [bodyMD, setBodyMD] = useState(initialBodyMDValue);
    const [cursorPos, setCursorPos] = useState(0);
    const { sendFlash } = useUserFlashCtx();
    const extraButtons = false;
    const handleChange = (_: React.SyntheticEvent, newValue: number) =>
        setTabSetTabValue(newValue);

    useEffect(() => {
        setBodyHTML(renderMarkdown(bodyMD));
    }, [bodyMD]);

    const onFileSave = useCallback(
        async (v: UserUploadedFile, onSuccess?: () => void) => {
            try {
                const resp = await apiSaveMedia(v);
                if (!resp.author_id) {
                    return;
                }
                const newBody =
                    bodyMD.slice(0, cursorPos) +
                    `![${resp.asset.name}](media://${resp.asset.asset_id})` +
                    bodyMD.slice(cursorPos);
                setBodyMD(newBody);
                onSuccess && onSuccess();
            } catch (e) {
                logErr(e);
                sendFlash('error', 'Failed to save media');
            }
        },
        [bodyMD, cursorPos, sendFlash]
    );

    return (
        <Stack>
            <Box
                sx={{
                    borderBottom: 1,
                    borderColor: 'divider'
                }}
            >
                <Tabs
                    variant={'fullWidth'}
                    value={setTabValue}
                    onChange={handleChange}
                    aria-label="Markdown & HTML Preview"
                >
                    <Tab label="Edit" />
                    <Tab label="Preview" color={'warning'} />
                </Tabs>
            </Box>
            <TabPanel value={setTabValue} index={0}>
                <Stack>
                    <Stack direction={'row'} alignItems={'center'} padding={2}>
                        <ButtonGroup>
                            <Tooltip title={'Insert image at current location'}>
                                <Button
                                    color="primary"
                                    aria-label="Upload Image Button"
                                    component="span"
                                    variant={'text'}
                                    onClick={async () => {
                                        const resp =
                                            await NiceModal.show<UserUploadedFile>(
                                                ModalFileUpload,
                                                {}
                                            );
                                        await onFileSave(resp);
                                    }}
                                    startIcon={<ImageIcon />}
                                >
                                    Insert Image
                                </Button>
                            </Tooltip>
                        </ButtonGroup>
                        {extraButtons && (
                            <ButtonGroup>
                                <Tooltip title={'Embolden selected text'}>
                                    <IconButton
                                        color="primary"
                                        aria-label="Bold"
                                        component="span"
                                    >
                                        <FormatBoldIcon />
                                    </IconButton>
                                </Tooltip>
                                <Tooltip title={'Underline selected text'}>
                                    <IconButton
                                        color="primary"
                                        aria-label="Underline"
                                        component="span"
                                    >
                                        <FormatUnderlinedIcon />
                                    </IconButton>
                                </Tooltip>
                                <Tooltip
                                    title={'Decrease indent of selected text'}
                                >
                                    <IconButton
                                        color="primary"
                                        aria-label="Decrease indent"
                                        component="span"
                                    >
                                        <FormatIndentDecreaseIcon />
                                    </IconButton>
                                </Tooltip>
                                <Tooltip
                                    title={'Increase indent of  selected text'}
                                >
                                    <IconButton
                                        color="primary"
                                        aria-label="Increase indent"
                                        component="span"
                                    >
                                        <FormatIndentIncreaseIcon />
                                    </IconButton>
                                </Tooltip>
                            </ButtonGroup>
                        )}
                    </Stack>
                    <Box paddingRight={2} paddingLeft={2}>
                        <TextField
                            sx={{
                                padding: 0,
                                minHeight: 350,
                                height: '100%'
                            }}
                            id="body"
                            label="Body (Markdown)"
                            fullWidth
                            multiline
                            rows={20}
                            value={bodyMD ?? ''}
                            onChange={(event) => {
                                const body = event.target.value;
                                setCursorPos(event.target.selectionEnd ?? 0);
                                setBodyMD(body);
                            }}
                        />
                    </Box>
                </Stack>
            </TabPanel>
            <TabPanel value={setTabValue} index={1}>
                <Box
                    padding={2}
                    sx={(theme) => {
                        return {
                            minHeight: 450,
                            a: {
                                color: theme.palette.text.primary
                            }
                        };
                    }}
                >
                    <article dangerouslySetInnerHTML={{ __html: bodyHTML }} />
                </Box>
            </TabPanel>
            <Box padding={2}>
                <ButtonGroup>
                    <Button
                        variant={'contained'}
                        color={'primary'}
                        onClick={() => {
                            if (bodyMD === '') {
                                sendFlash('error', 'Body cannot be empty');
                            } else {
                                onSave(bodyMD, () => {
                                    setBodyMD('');
                                });
                            }
                        }}
                    >
                        {saveLabel ?? 'Save'}
                    </Button>
                    {cancelEnabled && (
                        <Button
                            variant={'contained'}
                            color={'error'}
                            onClick={onCancel}
                        >
                            {cancelLabel ?? 'Cancel'}
                        </Button>
                    )}
                </ButtonGroup>
            </Box>
        </Stack>
    );
};
