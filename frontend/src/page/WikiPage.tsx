import React, { useCallback, useEffect, useMemo } from 'react';
import Grid from '@mui/material/Grid';
import { useParams } from 'react-router';
import Typography from '@mui/material/Typography';
import Stack from '@mui/material/Stack';
import { log } from '../util/errors';
import { LoadingSpinner } from '../component/LoadingSpinner';
import { apiGetWikiPage, apiSaveWikiPage, Page, renderWiki } from '../api/wiki';
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
    const [setTabValue, setTabSetTabValue] = React.useState(0);
    const [bodyHTML, setBodyHTML] = React.useState('');
    const [bodyMD, setBodyMD] = React.useState(initialBodyMDValue);
    const [title, setTitle] = React.useState(initialTitleValue);

    const handleChange = (_: React.SyntheticEvent, newValue: number) => {
        setTabSetTabValue(newValue);
    };

    return (
        <Stack>
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
                    <TextField
                        id="body"
                        label="Body (Markdown)"
                        fullWidth
                        multiline
                        minRows={15}
                        value={bodyMD ?? ''}
                        onChange={(event) => {
                            const body = event.target.value;
                            setBodyMD(body);
                            setBodyHTML(renderWiki(body));
                        }}
                    />
                </Stack>
            </TabPanel>
            <TabPanel value={setTabValue} index={1}>
                <Typography variant={'h3'}>{title}</Typography>
                <article dangerouslySetInnerHTML={{ __html: bodyHTML }} />
            </TabPanel>
            <ButtonGroup>
                <Button
                    variant={'outlined'}
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
        [page, flashes, setFlashes]
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
                    <LoadingSpinner />
                </Grid>
            )}
            {!loading && !editMode && page.revision > 0 && (
                <Grid item xs={12}>
                    <Typography variant={'h3'}>{page.title}</Typography>
                    <article dangerouslySetInnerHTML={{ __html: bodyHTML }} />
                    {currentUser.permission_level >=
                        PermissionLevel.Moderator && (
                        <ButtonGroup>
                            <Button
                                variant={'text'}
                                onClick={() => {
                                    setEditMode(true);
                                }}
                            >
                                Edit
                            </Button>
                        </ButtonGroup>
                    )}
                </Grid>
            )}
            {!loading && !editMode && page.revision == 0 && (
                <Grid item xs={12}>
                    <Stack spacing={3}>
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
                                    color={'success'}
                                    onClick={() => {
                                        setEditMode(true);
                                    }}
                                >
                                    Create It
                                </Button>
                            </Typography>
                        )}
                    </Stack>
                </Grid>
            )}
            {!loading && editMode && (
                <Grid item xs={12}>
                    <WikiEditForm
                        initialTitleValue={page.title}
                        initialBodyMDValue={page.body_md}
                        onSave={onSave}
                    />
                </Grid>
            )}
        </Grid>
    );
};
