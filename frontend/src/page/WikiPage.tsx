import React, { useCallback, useEffect, useMemo, useState } from 'react';
import Grid from '@mui/material/Grid';
import { useParams } from 'react-router';
import Typography from '@mui/material/Typography';
import Stack from '@mui/material/Stack';
import { log } from '../util/errors';
import { LoadingSpinner } from '../component/LoadingSpinner';
import {
    apiGetWikiPage,
    apiSaveWikiMedia,
    apiSaveWikiPage,
    Page,
    renderWiki
} from '../api/wiki';
import Button from '@mui/material/Button';
import Box from '@mui/material/Box';
import Tabs from '@mui/material/Tabs';
import Tab from '@mui/material/Tab';
import { TabPanel } from '../component/TabPanel';
import TextField from '@mui/material/TextField';
import ButtonGroup from '@mui/material/ButtonGroup';
import { useCurrentUserCtx } from '../contexts/CurrentUserCtx';
import { PermissionLevel } from '../api';
import { useUserFlashCtx } from '../contexts/UserFlashCtx';
import Paper from '@mui/material/Paper';
import EditIcon from '@mui/icons-material/Edit';
import IconButton from '@mui/material/IconButton';
import ImageIcon from '@mui/icons-material/Image';
import FormatBoldIcon from '@mui/icons-material/FormatBold';
import FormatUnderlinedIcon from '@mui/icons-material/FormatUnderlined';
import FormatIndentIncreaseIcon from '@mui/icons-material/FormatIndentIncrease';
import FormatIndentDecreaseIcon from '@mui/icons-material/FormatIndentDecrease';
import { Tooltip } from '@mui/material';
import { FileUploadModal } from '../component/FileUploadModal';

const defaultPage: Page = {
    slug: '',
    body_md: '',
    created_on: new Date(),
    updated_on: new Date(),
    revision: 0,
    title: ''
};

interface WikiEditFormProps {
    initialBodyMDValue: string;
    initialTitleValue: string;
    onSave: (title: string, body_md: string) => void;
}

const WikiEditForm = ({
    onSave,
    initialBodyMDValue,
    initialTitleValue
}: WikiEditFormProps): JSX.Element => {
    const [setTabValue, setTabSetTabValue] = useState(0);
    const [bodyHTML, setBodyHTML] = useState('');
    const [bodyMD, setBodyMD] = useState(initialBodyMDValue);
    const [title, setTitle] = useState(initialTitleValue);
    const [open, setOpen] = useState(false);
    const [cursorPos, setCursorPos] = useState(0);

    const handleChange = (_: React.SyntheticEvent, newValue: number) =>
        setTabSetTabValue(newValue);

    useEffect(() => {
        setBodyHTML(renderWiki(bodyMD));
    }, [bodyMD]);

    return (
        <Stack>
            <FileUploadModal
                open={open}
                setOpen={setOpen}
                onSave={(v) => {
                    apiSaveWikiMedia(v).then((resp) => {
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
            <Box sx={{ borderBottom: 1, borderColor: 'divider' }}>
                <Tabs
                    value={setTabValue}
                    onChange={handleChange}
                    aria-label="Markdown & HTML Preview"
                >
                    <Tab label="Edit" />
                    <Tab label="Preview" />
                </Tabs>
            </Box>
            <TabPanel value={setTabValue} index={0}>
                <Stack spacing={3}>
                    <TextField
                        id="title"
                        label="Title"
                        fullWidth
                        value={title ?? ''}
                        onChange={(event) => {
                            const title = event.target.value;
                            setTitle(title);
                        }}
                    />
                    <Stack direction={'row'} alignItems={'center'}>
                        <Typography variant={'subtitle1'}>Upload</Typography>
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
                        <Typography variant={'subtitle1'}>Format</Typography>
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
                    <TextField
                        id="body"
                        label="Body (Markdown)"
                        fullWidth
                        multiline
                        minRows={15}
                        value={bodyMD ?? ''}
                        onChange={(event) => {
                            const body = event.target.value;
                            setCursorPos(event.target.selectionEnd ?? 0);
                            setBodyMD(body);
                        }}
                    />
                </Stack>
            </TabPanel>
            <TabPanel value={setTabValue} index={1}>
                <Box
                    sx={(theme) => {
                        return {
                            a: {
                                color: theme.palette.text.primary
                            }
                        };
                    }}
                >
                    <Typography variant={'h1'}>{title}</Typography>
                    <article dangerouslySetInnerHTML={{ __html: bodyHTML }} />
                </Box>
            </TabPanel>
            <Box padding={3}>
                <ButtonGroup>
                    <Button
                        variant={'contained'}
                        color={'primary'}
                        onClick={() => {
                            if (title === '' || bodyMD === '') {
                                alert('Title and Body cannot be empty');
                            } else {
                                onSave(title, bodyMD);
                            }
                        }}
                    >
                        Save
                    </Button>
                </ButtonGroup>
            </Box>
        </Stack>
    );
};

export const WikiPage = (): JSX.Element => {
    const [page, setPage] = React.useState<Page>(defaultPage);
    const [loading, setLoading] = React.useState<boolean>(true);
    const [editMode, setEditMode] = React.useState<boolean>(false);
    const { slug } = useParams();
    const { currentUser } = useCurrentUserCtx();
    const { flashes, setFlashes } = useUserFlashCtx();

    useEffect(() => {
        setLoading(true);
        apiGetWikiPage(slug || 'home')
            .then((page) => {
                setPage(page);
            })
            .catch((e) => {
                log(e);
            });
        setLoading(false);
    }, [slug]);

    const onSave = useCallback(
        (new_title: string, new_body_md: string) => {
            const newPage = page;
            newPage.slug = slug || 'home';
            newPage.title = new_title;
            newPage.body_md = new_body_md;
            apiSaveWikiPage(newPage)
                .then((p) => {
                    setPage(p);
                    setFlashes([
                        ...flashes,
                        {
                            heading: 'Saved wiki page',
                            level: 'success',
                            message: `Slug ${p.slug} updated`,
                            closable: true
                        }
                    ]);
                    setEditMode(false);
                })
                .catch((e) => {
                    log(e);
                });
        },
        [page, slug, setFlashes, flashes]
    );

    const bodyHTML = useMemo(() => {
        return page.revision > 0 && page.body_md
            ? renderWiki(page.body_md)
            : '';
    }, [page.body_md, page.revision]);

    return (
        <Grid container paddingTop={3} spacing={3}>
            {loading && (
                <Grid item xs={12} alignContent={'center'}>
                    <Paper elevation={1}>
                        <LoadingSpinner />
                    </Paper>
                </Grid>
            )}
            {!loading && !editMode && page.revision > 0 && (
                <Grid item xs={12}>
                    <Paper elevation={1}>
                        <Stack padding={3}>
                            <Typography variant={'h1'}>{page.title}</Typography>
                            <Box
                                sx={(theme) => {
                                    return {
                                        img: {
                                            maxWidth: '100%'
                                        },
                                        a: {
                                            color: theme.palette.text.primary
                                        }
                                    };
                                }}
                                dangerouslySetInnerHTML={{ __html: bodyHTML }}
                            />
                            {currentUser.permission_level >=
                                PermissionLevel.Moderator && (
                                <ButtonGroup>
                                    <Button
                                        variant={'contained'}
                                        color={'primary'}
                                        onClick={() => {
                                            setEditMode(true);
                                        }}
                                        startIcon={<EditIcon />}
                                    >
                                        Edit Page
                                    </Button>
                                </ButtonGroup>
                            )}
                        </Stack>
                    </Paper>
                </Grid>
            )}
            {!loading && !editMode && page.revision == 0 && (
                <Grid item xs={12}>
                    <Paper elevation={1}>
                        <Stack spacing={3} padding={3}>
                            <Typography variant={'h1'}>
                                Wiki Entry Not Found
                            </Typography>
                            <Typography variant={'h3'}>
                                slug: {slug || 'home'}
                            </Typography>
                            {currentUser.permission_level >=
                                PermissionLevel.Moderator && (
                                <Typography variant={'body1'}>
                                    <Button
                                        variant={'contained'}
                                        color={'primary'}
                                        onClick={() => {
                                            setEditMode(true);
                                        }}
                                    >
                                        Create It
                                    </Button>
                                </Typography>
                            )}
                        </Stack>
                    </Paper>
                </Grid>
            )}
            {!loading && editMode && (
                <Grid item xs={12}>
                    <Paper elevation={1}>
                        <WikiEditForm
                            initialTitleValue={page.title}
                            initialBodyMDValue={page.body_md}
                            onSave={onSave}
                        />
                    </Paper>
                </Grid>
            )}
        </Grid>
    );
};
