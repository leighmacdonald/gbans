import React, { useState, useCallback } from 'react';
import NiceModal from '@ebay/nice-modal-react';
import EditIcon from '@mui/icons-material/Edit';
import FormatBoldIcon from '@mui/icons-material/FormatBold';
import FormatIndentDecreaseIcon from '@mui/icons-material/FormatIndentDecrease';
import FormatIndentIncreaseIcon from '@mui/icons-material/FormatIndentIncrease';
import FormatUnderlinedIcon from '@mui/icons-material/FormatUnderlined';
import ImageIcon from '@mui/icons-material/Image';
import PreviewIcon from '@mui/icons-material/Preview';
import Box from '@mui/material/Box';
import Button from '@mui/material/Button';
import ButtonGroup from '@mui/material/ButtonGroup';
import IconButton from '@mui/material/IconButton';
import Stack from '@mui/material/Stack';
import Tab from '@mui/material/Tab';
import Tabs from '@mui/material/Tabs';
import TextField from '@mui/material/TextField';
import Tooltip from '@mui/material/Tooltip';
import { useFormikContext } from 'formik';
import * as yup from 'yup';
import { apiSaveMedia, UserUploadedFile } from '../api/media';
import { useUserFlashCtx } from '../contexts/UserFlashCtx';
import { logErr } from '../util/errors';
import { MarkDownRenderer } from './MarkdownRenderer';
import { TabPanel } from './TabPanel';
import { ModalFileUpload } from './modal';

export const bodyMDValidator = yup.string().min(3, 'Message to short');

interface BodyMDFieldProps {
    body_md: string;
}

export const MDBodyField = <T,>({
    fileUpload = true
}: {
    fileUpload?: boolean;
}) => {
    const [setTabValue, setTabSetTabValue] = useState(0);
    const [cursorPos, setCursorPos] = useState(0);
    const { values, touched, setFieldValue, errors } = useFormikContext<
        T & BodyMDFieldProps
    >();
    const { sendFlash } = useUserFlashCtx();
    const extraButtons = false;

    const handleTabChange = (_: React.SyntheticEvent, newValue: number) =>
        setTabSetTabValue(newValue);

    const onFileSave = useCallback(
        async (v: UserUploadedFile, onSuccess?: () => void) => {
            try {
                const resp = await apiSaveMedia(v);
                if (!resp.author_id) {
                    return;
                }
                const newBody =
                    values.body_md.slice(0, cursorPos) +
                    `![${resp.asset.name}](media://${resp.asset.asset_id})` +
                    values.body_md.slice(cursorPos);
                await setFieldValue('body_md', newBody);
                onSuccess && onSuccess();
            } catch (e) {
                logErr(e);
                sendFlash('error', 'Failed to save media');
            }
        },
        [cursorPos, sendFlash, setFieldValue, values.body_md]
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
                    variant={'standard'}
                    value={setTabValue}
                    onChange={handleTabChange}
                    aria-label="Markdown & HTML Preview"
                >
                    <Tab
                        label="Edit"
                        icon={<EditIcon />}
                        iconPosition={'start'}
                    />
                    <Tab
                        label="Preview"
                        color={'warning'}
                        icon={<PreviewIcon />}
                        iconPosition={'start'}
                    />
                </Tabs>
            </Box>
            <TabPanel value={setTabValue} index={0}>
                <Stack>
                    {fileUpload && (
                        <Stack
                            direction={'row'}
                            alignItems={'center'}
                            padding={2}
                        >
                            <ButtonGroup>
                                <Tooltip
                                    title={'Insert image at current location'}
                                >
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
                                        title={
                                            'Decrease indent of selected text'
                                        }
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
                                        title={
                                            'Increase indent of  selected text'
                                        }
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
                    )}
                    <Box>
                        <TextField
                            sx={{
                                padding: 0,
                                minHeight: 350,
                                height: '100%'
                            }}
                            id="body_md"
                            name={'body_md'}
                            label="Body (Markdown)"
                            fullWidth
                            multiline
                            rows={20}
                            value={values.body_md}
                            error={touched.body_md && Boolean(errors.body_md)}
                            helperText={
                                touched.body_md &&
                                errors.body_md &&
                                `${errors.body_md}`
                            }
                            onChange={async (event) => {
                                const body = event.target.value;
                                setCursorPos(event.target.selectionEnd ?? 0);
                                await setFieldValue('body_md', body);
                            }}
                        />
                    </Box>
                </Stack>
            </TabPanel>
            <TabPanel value={setTabValue} index={1}>
                <Box padding={2}>
                    <MarkDownRenderer body_md={values.body_md} />
                </Box>
            </TabPanel>
        </Stack>
    );
};
