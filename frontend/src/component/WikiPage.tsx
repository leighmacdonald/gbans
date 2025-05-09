import { useMemo, useState } from 'react';
import ArticleIcon from '@mui/icons-material/Article';
import BuildIcon from '@mui/icons-material/Build';
import EditIcon from '@mui/icons-material/Edit';
import Button from '@mui/material/Button';
import ButtonGroup from '@mui/material/ButtonGroup';
import Grid from '@mui/material/Grid';
import MenuItem from '@mui/material/MenuItem';
import { useMutation, useQueryClient } from '@tanstack/react-query';
import { useLoaderData, useRouteContext } from '@tanstack/react-router';
import { z } from 'zod';
import { PermissionLevel, PermissionLevelCollection, permissionLevelString } from '../api';
import { apiSaveWikiPage, Page } from '../api/wiki.ts';
import { useAppForm } from '../contexts/formContext.tsx';
import { useUserFlashCtx } from '../hooks/useUserFlashCtx.ts';
import { ContainerWithHeaderAndButtons } from './ContainerWithHeaderAndButtons.tsx';
import { MarkDownRenderer } from './MarkdownRenderer.tsx';
import { mdEditorRef } from './field/MarkdownField.tsx';

interface WikiValues {
    body_md: string;
    permission_level: PermissionLevel;
}

export const WikiPage = ({ slug = 'home', path }: { slug: string; path: '/_guest/wiki/' | '/_guest/wiki/$slug' }) => {
    const [editMode, setEditMode] = useState<boolean>(false);
    const queryClient = useQueryClient();
    const { hasPermission } = useRouteContext({ from: path });
    const { sendFlash, sendError } = useUserFlashCtx();
    const page = useLoaderData({ from: path }) as Page;

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

    const mutation = useMutation({
        mutationKey: ['wiki', { slug }],
        mutationFn: async (values: WikiValues) => {
            const newPage: Page = {
                body_md: values.body_md,
                slug: slug ?? 'home',
                permission_level: values.permission_level,
                created_on: page?.created_on ?? new Date(),
                updated_on: page?.updated_on ?? new Date()
            };

            return await apiSaveWikiPage(newPage);
        },
        onSuccess: (savedPage) => {
            queryClient.setQueryData(['wiki', { slug }], savedPage);
            setEditMode(false);
            mdEditorRef.current?.setMarkdown('');
            sendFlash('success', `Updated ${slug} successfully. Revision: ${savedPage.revision}`);
        },
        onError: sendError
    });

    const form = useAppForm({
        onSubmit: async ({ value }) => {
            mutation.mutate(value);
        },
        validators: {
            onChange: z.object({
                permission_level: z.nativeEnum(PermissionLevel),
                body_md: z.string()
            })
        },
        defaultValues: {
            permission_level: page?.permission_level ?? PermissionLevel.Guest,
            body_md: page?.body_md ?? ''
        }
    });

    if (editMode) {
        return (
            <ContainerWithHeaderAndButtons title={`Editing: ${slug}`} iconLeft={<EditIcon />}>
                <form
                    onSubmit={async (e) => {
                        e.preventDefault();
                        e.stopPropagation();
                        await form.handleSubmit();
                    }}
                >
                    <Grid container spacing={2}>
                        <Grid size={{ xs: 12 }}>
                            <form.AppField
                                name={'permission_level'}
                                children={(field) => {
                                    return (
                                        <field.SelectField
                                            label={'Permissions'}
                                            items={PermissionLevelCollection}
                                            renderItem={(pl) => {
                                                return (
                                                    <MenuItem value={pl} key={`pl-${pl}`}>
                                                        {permissionLevelString(pl)}
                                                    </MenuItem>
                                                );
                                            }}
                                        />
                                    );
                                }}
                            />
                        </Grid>

                        <Grid size={{ xs: 12 }}>
                            <form.AppField
                                name={'body_md'}
                                children={(field) => {
                                    return <field.MarkdownField label={'Region'} />;
                                }}
                            />
                        </Grid>
                        <Grid size={{ xs: 12 }}>
                            <form.AppForm>
                                <form.CloseButton
                                    onClick={async () => {
                                        setEditMode(false);
                                    }}
                                />
                                <form.ResetButton />
                                <form.SubmitButton />
                            </form.AppForm>
                        </Grid>
                    </Grid>
                </form>
            </ContainerWithHeaderAndButtons>
        );
    }
    return (
        <Grid container spacing={2}>
            <Grid size={{ xs: editMode ? 6 : 12 }}>
                <ContainerWithHeaderAndButtons title={page?.slug ?? ''} iconLeft={<ArticleIcon />} buttons={buttons}>
                    <MarkDownRenderer body_md={page?.body_md ?? ''} />
                </ContainerWithHeaderAndButtons>
            </Grid>
        </Grid>
    );
};
