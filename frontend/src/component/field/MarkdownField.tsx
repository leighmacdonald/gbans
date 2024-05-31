import { useState, SyntheticEvent, useCallback } from 'react';
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
import { Asset } from '../../api/media.ts';
import { useUserFlashCtx } from '../../hooks/useUserFlashCtx.ts';
import { MarkDownRenderer } from '../MarkdownRenderer.tsx';
import { TabPanel } from '../TabPanel.tsx';
import { ModalFileUpload } from '../modal';
import { FieldProps } from './common.ts';

type MDBodyFieldProps = {
    fileUpload?: boolean;
    minHeight?: number;
    rows?: number;
} & FieldProps;

export const MarkdownField = ({
    state,
    handleChange,
    handleBlur,
    minHeight,
    rows = 20,
    fileUpload = true
}: MDBodyFieldProps) => {
    const { sendFlash } = useUserFlashCtx();
    const [setTabValue, setTabSetTabValue] = useState(0);
    const extraButtons = false;
    const [cursorPos, setCursorPos] = useState(0);

    const handleTabChange = (_: SyntheticEvent, newValue: number) => setTabSetTabValue(newValue);

    const onFileSave = useCallback(
        async (v: Asset, onSuccess?: () => void) => {
            try {
                const newBody =
                    state.value.slice(0, cursorPos) +
                    `![${v.name}](media://${v.asset_id})` +
                    state.value.slice(cursorPos);
                handleChange(() => {
                    return newBody;
                });
                onSuccess && onSuccess();
            } catch (e) {
                sendFlash('error', `Failed to save media: ${e}`);
            }
        },
        [cursorPos, handleChange, sendFlash, state.value]
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
                    <Tab label="Edit" icon={<EditIcon />} iconPosition={'start'} />
                    <Tab label="Preview" color={'warning'} icon={<PreviewIcon />} iconPosition={'start'} />
                </Tabs>
            </Box>
            <TabPanel value={setTabValue} index={0}>
                <Stack>
                    {fileUpload && (
                        <Stack direction={'row'} alignItems={'center'} padding={2}>
                            <ButtonGroup>
                                <Tooltip title={'Insert image at current location'}>
                                    <Button
                                        color="primary"
                                        aria-label="Upload Image Button"
                                        component="span"
                                        variant={'text'}
                                        onClick={async () => {
                                            const asset = await NiceModal.show<Asset>(ModalFileUpload, {});
                                            await onFileSave(asset);
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
                                        <IconButton color="primary" aria-label="Bold" component="span">
                                            <FormatBoldIcon />
                                        </IconButton>
                                    </Tooltip>
                                    <Tooltip title={'Underline selected text'}>
                                        <IconButton color="primary" aria-label="Underline" component="span">
                                            <FormatUnderlinedIcon />
                                        </IconButton>
                                    </Tooltip>
                                    <Tooltip title={'Decrease indent of selected text'}>
                                        <IconButton color="primary" aria-label="Decrease indent" component="span">
                                            <FormatIndentDecreaseIcon />
                                        </IconButton>
                                    </Tooltip>
                                    <Tooltip title={'Increase indent of  selected text'}>
                                        <IconButton color="primary" aria-label="Increase indent" component="span">
                                            <FormatIndentIncreaseIcon />
                                        </IconButton>
                                    </Tooltip>
                                </ButtonGroup>
                            )}
                        </Stack>
                    )}
                    <>
                        <TextField
                            sx={{
                                padding: 0,
                                minHeight: 150,
                                height: '100%'
                            }}
                            label="Body (Markdown)"
                            fullWidth
                            multiline
                            rows={rows}
                            value={state.value}
                            error={state.meta.touchedErrors.length > 0}
                            helperText={state.meta.touchedErrors}
                            onChange={(e) => {
                                setCursorPos(e.target.selectionEnd ?? 0);
                                handleChange(e.target.value);
                            }}
                            onBlur={handleBlur}
                        />
                    </>
                </Stack>
            </TabPanel>
            <TabPanel value={setTabValue} index={1}>
                <Box padding={2}>
                    <MarkDownRenderer body_md={state.value} minHeight={minHeight} />
                </Box>
            </TabPanel>
        </Stack>
    );
};
