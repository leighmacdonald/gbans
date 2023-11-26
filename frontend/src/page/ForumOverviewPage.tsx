import React, { JSX, useCallback, useEffect, useState } from 'react';
import NiceModal from '@ebay/nice-modal-react';
import ChatIcon from '@mui/icons-material/Chat';
import Button from '@mui/material/Button';
import Stack from '@mui/material/Stack';
import Typography from '@mui/material/Typography';
import Grid from '@mui/material/Unstable_Grid2';
import {
    apiGetForumOverview,
    Forum,
    ForumCategory,
    ForumOverview
} from '../api/forum';
import { ContainerWithHeader } from '../component/ContainerWithHeader';
import { ForumRowLink } from '../component/ForumRowLink';
import { VCenteredElement } from '../component/Heading';
import {
    ModalForumCategoryEditor,
    ModalForumForumEditor
} from '../component/modal';
import { useUserFlashCtx } from '../contexts/UserFlashCtx';
import { logErr } from '../util/errors';

const CategoryBlock = ({ category }: { category: ForumCategory }) => {
    return (
        <ContainerWithHeader title={category.title}>
            <Stack>
                {category.forums.map((f) => {
                    return (
                        <Grid container key={`forum-${f.forum_id}`}>
                            <Grid xs={6}>
                                <Stack direction={'row'} spacing={1}>
                                    <VCenteredElement
                                        icon={<ChatIcon scale={4} />}
                                    />
                                    <ForumRowLink
                                        label={f.title}
                                        to={`/forums/${f.forum_id}`}
                                    />
                                </Stack>
                            </Grid>
                            <Grid xs={6}>
                                <Stack direction={'row'}>
                                    <Typography variant={'subtitle1'}>
                                        {f.count_threads}
                                    </Typography>
                                    <Typography variant={'subtitle1'}>
                                        {f.count_messages}
                                    </Typography>
                                </Stack>
                            </Grid>
                        </Grid>
                    );
                })}
            </Stack>
        </ContainerWithHeader>
    );
};

export const ForumOverviewPage = (): JSX.Element => {
    const [overview, setOverview] = useState<ForumOverview>();
    const { sendFlash } = useUserFlashCtx();

    useEffect(() => {
        const abortController = new AbortController();
        apiGetForumOverview(abortController)
            .then((resp) => {
                setOverview(resp);
            })
            .catch((e) => logErr(e));
        return () => abortController.abort();
    }, []);

    const onNewCategory = useCallback(async () => {
        try {
            await NiceModal.show<ForumCategory>(ModalForumCategoryEditor, {});
            sendFlash('success', 'Created new category successfully');
        } catch (e) {
            logErr(e);
        }
    }, [sendFlash]);

    const onNewForum = useCallback(async () => {
        try {
            await NiceModal.show<Forum>(ModalForumForumEditor, {});
            sendFlash('success', 'Created new forum successfully');
        } catch (e) {
            logErr(e);
        }
    }, [sendFlash]);

    return (
        <Grid container spacing={3}>
            <Grid xs={12}>
                <Typography variant={'h2'}>
                    {window.gbans.site_name} community
                </Typography>
            </Grid>
            <Grid xs={9}>
                <Stack spacing={3}>
                    {overview?.categories.map((cat) => {
                        return (
                            <CategoryBlock
                                category={cat}
                                key={`category-${cat.forum_category_id}`}
                            />
                        );
                    })}
                </Stack>
            </Grid>
            <Grid xs={3}>
                <Stack spacing={3}>
                    <ContainerWithHeader title={'Mod Tools'}>
                        <Button
                            onClick={onNewCategory}
                            variant={'contained'}
                            color={'success'}
                        >
                            New Category
                        </Button>
                        <Button
                            onClick={onNewForum}
                            variant={'contained'}
                            color={'success'}
                        >
                            New Forum
                        </Button>
                    </ContainerWithHeader>
                </Stack>
            </Grid>
        </Grid>
    );
};
