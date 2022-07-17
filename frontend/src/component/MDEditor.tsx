import React, { useEffect, useState } from 'react';
import { renderMarkdown } from '../api/wiki';
import Stack from '@mui/material/Stack';
import { FileUploadModal } from './FileUploadModal';
import Box from '@mui/material/Box';
import Tabs from '@mui/material/Tabs';
import Tab from '@mui/material/Tab';
import { TabPanel } from './TabPanel';
import TextField from '@mui/material/TextField';
import ButtonGroup from '@mui/material/ButtonGroup';
import Tooltip from '@mui/material/Tooltip';
import IconButton from '@mui/material/IconButton';
import ImageIcon from '@mui/icons-material/Image';
import FormatBoldIcon from '@mui/icons-material/FormatBold';
import FormatUnderlinedIcon from '@mui/icons-material/FormatUnderlined';
import FormatIndentDecreaseIcon from '@mui/icons-material/FormatIndentDecrease';
import FormatIndentIncreaseIcon from '@mui/icons-material/FormatIndentIncrease';
import Button from '@mui/material/Button';
import { apiSaveMedia } from '../api/media';
import useTheme from '@mui/material/styles/useTheme';

interface MDEditorProps {
    initialBodyMDValue: string;
    onSave: (body_md: string) => void;
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
}: MDEditorProps): JSX.Element => {
    const theme = useTheme();
    const [setTabValue, setTabSetTabValue] = useState(0);
    const [bodyHTML, setBodyHTML] = useState('');
    const [bodyMD, setBodyMD] = useState(initialBodyMDValue);
    const [open, setOpen] = useState(false);
    const [cursorPos, setCursorPos] = useState(0);

    const handleChange = (_: React.SyntheticEvent, newValue: number) =>
        setTabSetTabValue(newValue);

    useEffect(() => {
        setBodyHTML(renderMarkdown(bodyMD));
    }, [bodyMD]);

    return (
        <Stack>
            <FileUploadModal
                open={open}
                setOpen={setOpen}
                onSave={(v) => {
                    apiSaveMedia(v).then((resp) => {
                        if (!resp.author_id) {
                            return;
                        }
                        setOpen(false);
                        const newBody =
                            bodyMD.slice(0, cursorPos) +
                            `![${resp.name}](media://${resp.name})` +
                            bodyMD.slice(cursorPos);
                        setBodyMD(newBody);
                    });
                }}
            />
            <Box
                sx={{
                    borderBottom: 1,
                    borderColor: 'divider',
                    backgroundColor: theme.palette.background.paper
                }}
            >
                <Tabs
                    variant={'fullWidth'}
                    value={setTabValue}
                    onChange={handleChange}
                    aria-label="Markdown & HTML Preview"
                >
                    <Tab label="Edit" />
                    <Tab label="Preview" />
                </Tabs>
            </Box>
            <TabPanel value={setTabValue} index={0}>
                <Stack>
                    <Stack direction={'row'} alignItems={'center'} padding={1}>
                        <ButtonGroup>
                            <Tooltip title={'Insert Image'}>
                                <IconButton
                                    color="primary"
                                    aria-label="Insert Image Button"
                                    component="span"
                                    onClick={() => setOpen(true)}
                                >
                                    <ImageIcon />
                                </IconButton>
                            </Tooltip>
                        </ButtonGroup>
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
                            <Tooltip title={'Decrease indent of selected text'}>
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
                                alert('Title and Body cannot be empty');
                            } else {
                                onSave(bodyMD);
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
