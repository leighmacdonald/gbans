import { ChangeEvent, ClipboardEvent, useCallback, useState, JSX } from 'react';
import NiceModal, { muiDialogV5, useModal } from '@ebay/nice-modal-react';
import { Dialog, DialogActions, DialogContent, DialogTitle, Divider } from '@mui/material';
import Button from '@mui/material/Button';
import LinearProgress from '@mui/material/LinearProgress';
import Paper from '@mui/material/Paper';
import Stack from '@mui/material/Stack';
import Typography from '@mui/material/Typography';
import Grid from '@mui/material/Unstable_Grid2';
import { useTheme } from '@mui/material/styles';
import { styled } from '@mui/material/styles';
import { useForm } from '@tanstack/react-form';
import { useMutation } from '@tanstack/react-query';
import { zodValidator } from '@tanstack/zod-form-adapter';
import { fromByteArray } from 'base64-js';
import { apiSaveMedia, UserUploadedFile } from '../../api/media.ts';
import { useUserFlashCtx } from '../../hooks/useUserFlashCtx.ts';
import { humanFileSize } from '../../util/text.tsx';
import { Heading } from '../Heading';
import { Buttons } from '../field/Buttons.tsx';
import { TextFieldSimple } from '../field/TextFieldSimple.tsx';

const MethodPaper = styled(Paper)(({ theme }) => ({
    padding: theme.spacing(1),
    textAlign: 'center',
    lineHeight: '60px',
    minWidth: '30%'
}));

type FileUploadModalProps = {
    name: string;
    file: UserUploadedFile;
};

export const FileUploadModal = NiceModal.create((): JSX.Element => {
    const theme = useTheme();
    const { sendFlash } = useUserFlashCtx();
    const [upload, setUpload] = useState<UserUploadedFile>();
    const [progress, setProgress] = useState(0);
    const [progressTotal, setProgressTotal] = useState(100);
    const [uploadInProgress, setUploadInProgress] = useState(false);
    const [enabledPanel, setEnabledPanel] = useState<'all' | 'file' | 'url' | 'paste'>('all');
    const modal = useModal();

    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    const handleUploadedFile = useCallback(({ target }: ChangeEvent<HTMLInputElement>) => {
        if (!target.files) {
            return;
        }
        const file = target.files[0];
        const reader = new FileReader();
        reader.addEventListener('load', (event) => {
            console.log('loaded');
            console.log(event);
            if (event?.target?.result) {
                setUpload({
                    content: fromByteArray(new Uint8Array(event.target.result as ArrayBuffer)),
                    mime: file.type,
                    name: file.name,
                    size: file.size
                });
                setEnabledPanel('file');
            }
        });

        reader.readAsArrayBuffer(file);
    }, []);

    const handlePaste = useCallback((event: ClipboardEvent) => {
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
                        const content = fromByteArray(new Uint8Array(event.target.result as ArrayBuffer));
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
            } else {
                sendFlash('error', 'Invalid paste type, must copy file/image');
            }
        }
    }, []);

    const mutation = useMutation({
        mutationKey: ['uploadAsset'],
        mutationFn: async ({ name, file }: FileUploadModalProps) => {
            if (name != '') {
                file.name = name;
            }
            return await apiSaveMedia(file);
        },
        onSuccess: async (media) => {
            modal.resolve(media);
            await modal.hide();
        },
        onError: (error) => {
            sendFlash('error', `Failed to upload file: ${error}`);
        }
    });

    const { Field, Subscribe, handleSubmit, reset } = useForm({
        onSubmit: async ({ value }) => {
            if (!upload?.content) {
                throw 'No file uploaded';
            }
            mutation.mutate({
                name: value.name,
                file: upload
            });
        },
        validatorAdapter: zodValidator,
        defaultValues: {
            name: ''
        }
    });

    const resetAll = async () => {
        setEnabledPanel('all');
        setProgress(0);
        setUploadInProgress(false);
        setUpload(undefined);
        reset();
    };

    return (
        <Dialog
            aria-labelledby="modal-modal-title"
            aria-describedby="modal-modal-description"
            onPaste={handlePaste}
            fullWidth
            maxWidth={'lg'}
            {...muiDialogV5(modal)}
        >
            <form
                onSubmit={async (e) => {
                    e.preventDefault();
                    e.stopPropagation();
                    await handleSubmit();
                }}
            >
                <DialogTitle component={Heading}>Upload An Image</DialogTitle>
                <DialogContent>
                    <Grid container spacing={2}>
                        <Grid xs={12}>
                            <Typography id="modal-modal-description" sx={{ mt: 2 }}>
                                You can upload evidence screenshots by choosing one of the 3 methods below.
                            </Typography>
                        </Grid>
                        <Grid xs={12}>
                            <Stack
                                direction={{ xs: 'column', sm: 'row' }}
                                justifyContent="space-evenly"
                                alignItems="stretch"
                                divider={<Divider orientation="vertical" flexItem />}
                                spacing={2}
                            >
                                <MethodPaper elevation={1}>
                                    <Typography variant={'subtitle1'}>File Upload</Typography>
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
                                            disabled={!['all', 'file'].includes(enabledPanel)}
                                        >
                                            Select File
                                        </Button>
                                    </label>
                                </MethodPaper>

                                <MethodPaper>
                                    <Typography variant={'subtitle1'}>Paste</Typography>
                                    <Typography
                                        variant={'body2'}
                                        color={'disabled'}
                                        sx={{
                                            color: !['all', 'paste'].includes(enabledPanel)
                                                ? theme.palette.grey['500']
                                                : theme.typography.body2.color
                                        }}
                                    >
                                        You can capture a screen shot (Windows screenshot shortcut:{' '}
                                        <kbd>win+shift+s</kbd>) and paste it anywhere in the window using{' '}
                                        <kbd>ctrl+v</kbd>.
                                    </Typography>
                                </MethodPaper>
                            </Stack>
                        </Grid>
                        <Grid xs={12}>
                            <Field
                                name={'name'}
                                children={(props) => {
                                    return <TextFieldSimple {...props} label={'Name'} />;
                                }}
                            />
                        </Grid>
                        <Grid xs={12}>
                            <Stack
                                spacing={2}
                                direction="row-reverse"
                                justifyContent="flex-start"
                                alignItems="flex-start"
                            >
                                {uploadInProgress && <LinearProgress value={progress} valueBuffer={progressTotal} />}
                            </Stack>
                        </Grid>
                        {upload?.size && (
                            <Grid xs={12}>
                                <Stack direction={'row'} spacing={2}>
                                    <Typography>Name: {upload.name}</Typography>{' '}
                                    <Typography>Mimetype: {upload.mime}</Typography>{' '}
                                    <Typography>Size: {humanFileSize(upload.size)}</Typography>
                                </Stack>
                            </Grid>
                        )}
                    </Grid>
                </DialogContent>
                <DialogActions>
                    <Grid container>
                        <Grid xs={12} mdOffset="auto">
                            <Subscribe
                                selector={(state) => [state.canSubmit, state.isSubmitting]}
                                children={([canSubmit, isSubmitting]) => {
                                    return (
                                        <Buttons
                                            submitLabel={'Upload File'}
                                            reset={resetAll}
                                            canSubmit={canSubmit}
                                            isSubmitting={isSubmitting}
                                            onClose={async () => {
                                                await modal.hide();
                                            }}
                                        />
                                    );
                                }}
                            />
                        </Grid>
                    </Grid>
                </DialogActions>
            </form>
        </Dialog>
    );
});

export default FileUploadModal;
