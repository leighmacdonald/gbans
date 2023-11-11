import React, { useCallback, useEffect, useMemo, JSX } from 'react';
import { useParams } from 'react-router';
import ArticleIcon from '@mui/icons-material/Article';
import Box from '@mui/material/Box';
import Button from '@mui/material/Button';
import Paper from '@mui/material/Paper';
import Typography from '@mui/material/Typography';
import Grid from '@mui/material/Unstable_Grid2';
import { PermissionLevel } from '../api';
import {
    apiGetWikiPage,
    apiSaveWikiPage,
    Page,
    renderMarkdown
} from '../api/wiki';
import { ContainerWithHeader } from '../component/ContainerWithHeader';
import { LoadingSpinner } from '../component/LoadingSpinner';
import { MDEditor } from '../component/MDEditor';
import { RenderedMarkdownBox } from '../component/RenderedMarkdownBox';
import { useCurrentUserCtx } from '../contexts/CurrentUserCtx';
import { useUserFlashCtx } from '../contexts/UserFlashCtx';
import { logErr } from '../util/errors';

const defaultPage: Page = {
    slug: '',
    body_md: '',
    created_on: new Date(),
    updated_on: new Date(),
    revision: 0,
    title: ''
};

export const WikiPage = (): JSX.Element => {
    const [page, setPage] = React.useState<Page>(defaultPage);
    const [loading, setLoading] = React.useState<boolean>(true);
    const [editMode, setEditMode] = React.useState<boolean>(false);
    const { slug } = useParams();
    const { currentUser } = useCurrentUserCtx();
    const { sendFlash } = useUserFlashCtx();

    useEffect(() => {
        const abortController = new AbortController();
        setLoading(true);
        const fetchWiki = async () => {
            try {
                const page = await apiGetWikiPage(
                    slug || 'home',
                    abortController
                );
                setPage(page);
            } catch (e) {
                logErr(e);
            } finally {
                setLoading(false);
            }
        };

        fetchWiki().catch(logErr);

        return () => abortController.abort();
    }, [slug]);

    const onSave = useCallback(
        (new_body_md: string) => {
            const newPage = page;
            newPage.slug = slug || 'home';
            newPage.body_md = new_body_md;
            apiSaveWikiPage(newPage)
                .then((response) => {
                    setPage(response);
                    sendFlash('success', `Slug ${response.slug} updated`);
                    setEditMode(false);
                })
                .catch(logErr);
        },
        [page, sendFlash, slug]
    );

    const bodyHTML = useMemo(() => {
        return page.revision > 0 && page.body_md
            ? renderMarkdown(page.body_md)
            : '';
    }, [page.body_md, page.revision]);

    return (
        <Grid container spacing={3}>
            {loading && (
                <Grid xs={12} alignContent={'center'}>
                    <Paper elevation={1}>
                        <LoadingSpinner />
                    </Paper>
                </Grid>
            )}
            {!loading && !editMode && page.revision > 0 && (
                <Grid xs={12}>
                    <ContainerWithHeader
                        title={page.slug}
                        iconLeft={<ArticleIcon />}
                    >
                        <Box padding={2}>
                            <RenderedMarkdownBox
                                bodyHTML={bodyHTML}
                                readonly={
                                    currentUser.permission_level <
                                    PermissionLevel.Moderator
                                }
                                setEditMode={setEditMode}
                            />
                        </Box>
                    </ContainerWithHeader>
                </Grid>
            )}
            {!loading && !editMode && page.revision == 0 && (
                <Grid xs={12}>
                    <ContainerWithHeader
                        title={'Wiki Entry Not Found'}
                        iconLeft={<ArticleIcon />}
                    >
                        <>
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
                        </>
                    </ContainerWithHeader>
                </Grid>
            )}
            {!loading && editMode && (
                <Grid xs={12}>
                    <Paper elevation={1}>
                        <MDEditor
                            initialBodyMDValue={page.body_md}
                            onSave={onSave}
                        />
                    </Paper>
                </Grid>
            )}
        </Grid>
    );
};
