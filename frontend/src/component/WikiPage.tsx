import { useMemo, useState } from 'react';
import ArticleIcon from '@mui/icons-material/Article';
import BuildIcon from '@mui/icons-material/Build';
import EditIcon from '@mui/icons-material/Edit';
import Button from '@mui/material/Button';
import ButtonGroup from '@mui/material/ButtonGroup';
import MenuItem from '@mui/material/MenuItem';
import Grid from '@mui/material/Unstable_Grid2';
import { useForm } from '@tanstack/react-form';
import { useMutation, useQueryClient } from '@tanstack/react-query';
import { useLoaderData, useRouteContext } from '@tanstack/react-router';
import { zodValidator } from '@tanstack/zod-form-adapter';
import { z } from 'zod';
import { PermissionLevel, PermissionLevelCollection, permissionLevelString } from '../api';
import { apiSaveWikiPage, Page } from '../api/wiki.ts';
import { useUserFlashCtx } from '../hooks/useUserFlashCtx.ts';
import { ContainerWithHeaderAndButtons } from './ContainerWithHeaderAndButtons.tsx';
import { MarkDownRenderer } from './MarkdownRenderer.tsx';
import { Buttons } from './field/Buttons.tsx';
import { MarkdownField } from './field/MarkdownField.tsx';
import { SelectFieldSimple } from './field/SelectFieldSimple.tsx';

interface WikiValues {
    body_md: string;
    permission_level: PermissionLevel;
}

export const WikiPage = ({ slug = 'home', path }: { slug: string; path: '/_guest/wiki/' | '/_guest/wiki/$slug' }) => {
    const [editMode, setEditMode] = useState<boolean>(false);
    const queryClient = useQueryClient();
    const { hasPermission } = useRouteContext({ from: path });
    const { sendFlash } = useUserFlashCtx();
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
                        setEditMode((prev) => !prev);
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
            sendFlash('success', `Updated ${slug} successfully. Revision: ${savedPage.revision}`);
        }
    });

    const { Field, Subscribe, handleSubmit, reset } = useForm({
        onSubmit: async ({ value }) => {
            mutation.mutate(value);
        },
        validatorAdapter: zodValidator,
        defaultValues: {
            permission_level: page?.permission_level ?? PermissionLevel.Guest,
            body_md: page?.body_md ?? ''
        }
    });

    const editButtons = (
        <Subscribe
            key={'edit-form-buttons'}
            selector={(state) => [state.canSubmit, state.isSubmitting]}
            children={([canSubmit, isSubmitting]) => {
                return (
                    <Buttons
                        key={'edit-buttons'}
                        reset={reset}
                        canSubmit={canSubmit}
                        isSubmitting={isSubmitting}
                        closeLabel={'Cancel'}
                        onClose={async () => {
                            setEditMode(false);
                        }}
                    />
                );
            }}
        />
    );

    if (editMode) {
        return (
            <form
                onSubmit={async (e) => {
                    e.preventDefault();
                    e.stopPropagation();
                    await handleSubmit();
                }}
            >
                <ContainerWithHeaderAndButtons
                    title={`Editing: ${slug}`}
                    buttons={[editButtons]}
                    iconLeft={<EditIcon />}
                >
                    <Grid container spacing={2}>
                        <Grid xs={12}>
                            <Field
                                name={'permission_level'}
                                validators={{
                                    onChange: z.nativeEnum(PermissionLevel)
                                }}
                                children={(props) => {
                                    return (
                                        <SelectFieldSimple
                                            {...props}
                                            label={'Permissions'}
                                            fullwidth={true}
                                            items={PermissionLevelCollection}
                                            renderMenu={(pl) => {
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

                        <Grid xs={12}>
                            <Field
                                name={'body_md'}
                                validators={{
                                    onChange: z.string().min(5)
                                }}
                                children={(props) => {
                                    return <MarkdownField {...props} label={'Region'} />;
                                }}
                            />
                        </Grid>
                    </Grid>
                    <Grid xs={12} mdOffset="auto"></Grid>
                </ContainerWithHeaderAndButtons>
            </form>
        );
    }
    return (
        <Grid container spacing={2}>
            <Grid xs={editMode ? 6 : 12}>
                <ContainerWithHeaderAndButtons title={page?.slug ?? ''} iconLeft={<ArticleIcon />} buttons={buttons}>
                    <MarkDownRenderer body_md={page?.body_md ?? ''} />
                </ContainerWithHeaderAndButtons>
            </Grid>
        </Grid>
    );
};
