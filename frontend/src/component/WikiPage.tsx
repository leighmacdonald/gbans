import { useCallback, useMemo, useState } from 'react';
import ArticleIcon from '@mui/icons-material/Article';
import BuildIcon from '@mui/icons-material/Build';
import Button from '@mui/material/Button';
import ButtonGroup from '@mui/material/ButtonGroup';
import Paper from '@mui/material/Paper';
import Stack from '@mui/material/Stack';
import Typography from '@mui/material/Typography';
import Grid from '@mui/material/Unstable_Grid2';
import { useRouteContext } from '@tanstack/react-router';
import { Formik } from 'formik';
import * as yup from 'yup';
import { ErrorCode, PermissionLevel } from '../api';
import { apiSaveWikiPage, Page } from '../api/wiki.ts';
import { useWiki } from '../hooks/useWiki.ts';
import { PageNotFound } from '../routes/_auth.page-not-found.lazy.tsx';
import { LoginPage } from '../routes/_guest.login.index.tsx';
import { logErr } from '../util/errors.ts';
import { bodyMDValidator } from '../util/validators.ts';
import { ContainerWithHeader } from './ContainerWithHeader.tsx';
import { ContainerWithHeaderAndButtons } from './ContainerWithHeaderAndButtons.tsx';
import { MDBodyField } from './MDBodyField.tsx';
import { MarkDownRenderer } from './MarkdownRenderer.tsx';
import { PermissionLevelField } from './formik/PermissionLevelField.tsx';
import { SubmitButton } from './modal/Buttons.tsx';

interface WikiValues {
    body_md: string;
    permission_level: PermissionLevel;
}

const validationSchema = yup.object({
    body_md: bodyMDValidator,
    permission_level: yup
        .number()
        .oneOf([
            PermissionLevel.Guest,
            PermissionLevel.User,
            PermissionLevel.Reserved,
            PermissionLevel.Editor,
            PermissionLevel.Moderator,
            PermissionLevel.Admin
        ])
        .label('Permission Level')
        .required('Minimum permission value required')
});

export const WikiPage = ({ slug = 'home' }: { slug: string }) => {
    const [updatedPage, setUpdatedPage] = useState<Page>();
    const [editMode, setEditMode] = useState<boolean>(false);
    const { hasPermission, permissionLevel } = useRouteContext({ from: '/_guest/wiki' });
    const { data, loading, error } = useWiki(slug);

    const isPermDenied = useMemo(() => {
        if (!error) {
            return false;
        }

        return error.code == ErrorCode.PermissionDenied;
    }, [error]);

    const page = useMemo(() => {
        return updatedPage ?? data;
    }, [data, updatedPage]);

    const onSubmit = useCallback(
        async (values: WikiValues) => {
            try {
                const newPage = {
                    ...page,
                    body_md: values.body_md,
                    slug: slug ?? 'home',
                    permission_level: values.permission_level
                };
                const resp = await apiSaveWikiPage(newPage);
                setUpdatedPage(resp);
                setEditMode(false);
            } catch (e) {
                logErr(e);
            }
        },
        [page, slug]
    );

    const bodyHTML = useMemo(() => {
        return page.revision > 0 && page.body_md ? <MarkDownRenderer body_md={page.body_md} /> : '';
    }, [page.body_md, page.revision]);

    const buttons = useMemo(() => {
        if (!hasPermission(PermissionLevel.Editor)) {
            return [];
        }
        return [
            <ButtonGroup key={`wiki-buttons`}>
                <Button
                    startIcon={<BuildIcon />}
                    variant={'contained'}
                    color={'warning'}
                    onClick={() => {
                        setEditMode(true);
                    }}
                >
                    Edit
                </Button>
            </ButtonGroup>
        ];
    }, [hasPermission]);

    return (
        <Grid container spacing={3}>
            {!loading && !editMode && page.revision > 0 && (
                <Grid xs={12}>
                    <ContainerWithHeaderAndButtons title={page.slug} iconLeft={<ArticleIcon />} buttons={buttons}>
                        {bodyHTML}
                    </ContainerWithHeaderAndButtons>
                </Grid>
            )}
            {!loading && !editMode && page.revision == 0 && !isPermDenied && (
                <Grid xs={12}>
                    <ContainerWithHeader title={'Wiki Entry Not Found'} iconLeft={<ArticleIcon />}>
                        <>
                            <Typography variant={'h3'}>slug: {slug || 'home'}</Typography>
                            {hasPermission(PermissionLevel.Moderator) && (
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
            {isPermDenied && hasPermission(PermissionLevel.Guest) && (
                <Grid xs={12}>
                    <PageNotFound />
                </Grid>
            )}
            {isPermDenied && permissionLevel() == PermissionLevel.Guest && (
                <Grid xs={12}>
                    <LoginPage />
                </Grid>
            )}
            {!loading && editMode && !isPermDenied && (
                <Grid xs={12}>
                    <Formik<WikiValues>
                        onSubmit={onSubmit}
                        validationSchema={validationSchema}
                        validateOnBlur={true}
                        initialValues={{
                            body_md: page.body_md,
                            permission_level: page.permission_level
                        }}
                    >
                        <Paper elevation={1}>
                            <Stack spacing={1} padding={1}>
                                <MDBodyField />
                                <PermissionLevelField
                                    levels={[
                                        PermissionLevel.Guest,
                                        PermissionLevel.User,
                                        PermissionLevel.Reserved,
                                        PermissionLevel.Editor,
                                        PermissionLevel.Moderator,
                                        PermissionLevel.Admin
                                    ]}
                                />
                                <ButtonGroup>
                                    <Button
                                        color={'warning'}
                                        variant={'contained'}
                                        onClick={() => {
                                            setEditMode(false);
                                        }}
                                    >
                                        Cancel
                                    </Button>
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
