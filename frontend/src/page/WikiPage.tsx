import React, { useCallback, useEffect, useMemo, JSX } from 'react';
import Grid from '@mui/material/Unstable_Grid2';
import { useParams } from 'react-router';
import Typography from '@mui/material/Typography';
import { log, logErr } from '../util/errors';
import { LoadingSpinner } from '../component/LoadingSpinner';
import {
    apiGetWikiPage,
    apiSaveWikiPage,
    Page,
    renderMarkdown
} from '../api/wiki';
import Button from '@mui/material/Button';
import { useCurrentUserCtx } from '../contexts/CurrentUserCtx';
import { PermissionLevel } from '../api';
import { useUserFlashCtx } from '../contexts/UserFlashCtx';
import Paper from '@mui/material/Paper';
import { MDEditor } from '../component/MDEditor';
import { RenderedMarkdownBox } from '../component/RenderedMarkdownBox';
import Box from '@mui/material/Box';
import { ContainerWithHeader } from '../component/ContainerWithHeader';
import ArticleIcon from '@mui/icons-material/Article';

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
        setLoading(true);
        apiGetWikiPage(slug || 'home')
            .then((response) => {
                if (!response.status || !response.result) {
                    sendFlash('error', 'Failed to load wiki page');
                    return;
                }
                setPage(response.result);
            })
            .catch((e) => {
                log(e);
            });
        setLoading(false);
    }, [sendFlash, slug]);

    const onSave = useCallback(
        (new_body_md: string) => {
            const newPage = page;
            newPage.slug = slug || 'home';
            newPage.body_md = new_body_md;
            apiSaveWikiPage(newPage)
                .then((response) => {
                    if (!response.status || !response.result) {
                        sendFlash('error', 'Failed to save wiki page');
                        return;
                    }
                    setPage(response.result);
                    sendFlash(
                        'success',
                        `Slug ${response.result.slug} updated`
                    );
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
        <Grid container paddingTop={3} spacing={3}>
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
