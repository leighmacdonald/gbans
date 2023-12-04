import React, { useEffect, useMemo, JSX, useCallback } from 'react';
import { useParams } from 'react-router';
import ArticleIcon from '@mui/icons-material/Article';
import Box from '@mui/material/Box';
import Button from '@mui/material/Button';
import ButtonGroup from '@mui/material/ButtonGroup';
import Paper from '@mui/material/Paper';
import Stack from '@mui/material/Stack';
import Typography from '@mui/material/Typography';
import Grid from '@mui/material/Unstable_Grid2';
import { Formik } from 'formik';
import * as yup from 'yup';
import { PermissionLevel } from '../api';
import { apiGetWikiPage, apiSaveWikiPage, Page } from '../api/wiki';
import { ContainerWithHeader } from '../component/ContainerWithHeader';
import { LoadingSpinner } from '../component/LoadingSpinner';
import { bodyMDValidator, MDBodyField } from '../component/MDBodyField';
import { MarkDownRenderer } from '../component/MarkdownRenderer';
import { SubmitButton } from '../component/modal/Buttons';
import { useCurrentUserCtx } from '../contexts/CurrentUserCtx';
import { logErr } from '../util/errors';

const defaultPage: Page = {
    slug: '',
    body_md: '',
    created_on: new Date(),
    updated_on: new Date(),
    revision: 0,
    title: ''
};

interface WikiValues {
    slug: string;
    body_md: string;
}

const validationSchema = yup.object({
    body_md: bodyMDValidator
});

export const WikiPage = (): JSX.Element => {
    const [page, setPage] = React.useState<Page>(defaultPage);
    const [loading, setLoading] = React.useState<boolean>(true);
    const [editMode, setEditMode] = React.useState<boolean>(false);
    const { slug } = useParams();
    const { currentUser } = useCurrentUserCtx();

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

    const onSubmit = useCallback(
        async (values: WikiValues) => {
            try {
                const newPage = {
                    ...page,
                    body_md: values.body_md,
                    slug: values.slug
                };
                const resp = await apiSaveWikiPage(newPage);
                setPage(resp);
                setEditMode(false);
            } catch (e) {
                logErr(e);
            }
        },
        [page]
    );

    const bodyHTML = useMemo(() => {
        return page.revision > 0 && page.body_md ? (
            <MarkDownRenderer body_md={page.body_md} />
        ) : (
            ''
        );
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
                        <Box padding={2}>{bodyHTML}</Box>
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
                    <Formik<WikiValues>
                        onSubmit={onSubmit}
                        validationSchema={validationSchema}
                        validateOnBlur={true}
                        initialValues={{
                            slug: page.slug,
                            body_md: page.body_md
                        }}
                    >
                        <Paper elevation={1}>
                            <Stack spacing={1} padding={1}>
                                <MDBodyField />
                                <ButtonGroup>
                                    <SubmitButton />
                                </ButtonGroup>
                            </Stack>
                        </Paper>
                    </Formik>
                </Grid>
            )}
        </Grid>
    );
};
