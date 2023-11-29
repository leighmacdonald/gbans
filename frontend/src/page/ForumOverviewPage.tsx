import React, { JSX, useCallback, useEffect, useState } from 'react';
import { Link as RouterLink } from 'react-router-dom';
import NiceModal from '@ebay/nice-modal-react';
import { Person2 } from '@mui/icons-material';
import AccessTimeIcon from '@mui/icons-material/AccessTime';
import ChatIcon from '@mui/icons-material/Chat';
import ConstructionIcon from '@mui/icons-material/Construction';
import TodayIcon from '@mui/icons-material/Today';
import Avatar from '@mui/material/Avatar';
import Button from '@mui/material/Button';
import Stack from '@mui/material/Stack';
import Typography from '@mui/material/Typography';
import Grid from '@mui/material/Unstable_Grid2';
import { defaultAvatarHash, PermissionLevel } from '../api';
import {
    apiGetForumOverview,
    Forum,
    ForumCategory,
    ForumOverview
} from '../api/forum';
import { ContainerWithHeader } from '../component/ContainerWithHeader';
import { ForumRowLink } from '../component/ForumRowLink';
import { VCenteredElement } from '../component/Heading';
import { VCenterBox } from '../component/VCenterBox';
import {
    ModalForumCategoryEditor,
    ModalForumForumEditor
} from '../component/modal';
import { useCurrentUserCtx } from '../contexts/CurrentUserCtx';
import { useUserFlashCtx } from '../contexts/UserFlashCtx';
import { logErr } from '../util/errors';
import { humanCount, renderDateTime } from '../util/text';

const CategoryBlock = ({ category }: { category: ForumCategory }) => {
    return (
        <ContainerWithHeader title={category.title}>
            <Stack spacing={1}>
                {category.forums.map((f) => {
                    return (
                        <Grid
                            container
                            key={`forum-${f.forum_id}`}
                            spacing={1}
                            sx={{
                                '&:hover': {
                                    backgroundColor: (theme) =>
                                        theme.palette.background.default
                                }
                            }}
                        >
                            <Grid xs={5} margin={0}>
                                <VCenterBox justify={'left'}>
                                    <Stack direction={'row'} spacing={1}>
                                        <VCenteredElement icon={<ChatIcon />} />
                                        <Stack>
                                            <ForumRowLink
                                                label={f.title}
                                                to={`/forums/${f.forum_id}`}
                                            />
                                            <Typography variant={'body2'}>
                                                {f.description}
                                            </Typography>
                                        </Stack>
                                    </Stack>
                                </VCenterBox>
                            </Grid>
                            <Grid xs={2}>
                                <Stack direction={'row'} spacing={1}>
                                    <Stack>
                                        <Typography
                                            variant={'body2'}
                                            align={'left'}
                                        >
                                            Threads
                                        </Typography>
                                        <Typography
                                            variant={'body1'}
                                            align={'center'}
                                        >
                                            {humanCount(f.count_threads)}
                                        </Typography>
                                    </Stack>
                                    <Stack>
                                        <Typography variant={'body2'}>
                                            Messages
                                        </Typography>
                                        <Typography
                                            variant={'body1'}
                                            align={'center'}
                                        >
                                            {humanCount(f.count_messages)}
                                        </Typography>
                                    </Stack>
                                </Stack>
                            </Grid>
                            <Grid xs={5}>
                                {f.recent_forum_thread_id &&
                                f.recent_forum_thread_id > 0 ? (
                                    <Stack direction={'row'} spacing={2}>
                                        <VCenteredElement
                                            icon={
                                                <Avatar
                                                    alt={f.recent_personaname}
                                                    src={`https://avatars.akamai.steamstatic.com/${
                                                        f.recent_avatarhash ??
                                                        defaultAvatarHash
                                                    }.jpg`}
                                                />
                                            }
                                        />
                                        <Stack>
                                            <ForumRowLink
                                                label={
                                                    f.recent_forum_title ?? ''
                                                }
                                                to={`/forums/thread/${f.recent_forum_thread_id}`}
                                            />

                                            <Stack
                                                direction={'row'}
                                                spacing={1}
                                            >
                                                <AccessTimeIcon />
                                                <VCenterBox>
                                                    <Typography
                                                        variant={'body2'}
                                                    >
                                                        {renderDateTime(
                                                            f.recent_created_on ??
                                                                new Date()
                                                        )}
                                                    </Typography>
                                                </VCenterBox>
                                                <Person2 />
                                                <VCenterBox>
                                                    <Typography
                                                        color={(theme) => {
                                                            return theme.palette
                                                                .text.secondary;
                                                        }}
                                                        component={RouterLink}
                                                        to={`/profile/${f.recent_source_id}`}
                                                        variant={'body2'}
                                                    >
                                                        {f.recent_personaname}
                                                    </Typography>
                                                </VCenterBox>
                                            </Stack>
                                        </Stack>
                                    </Stack>
                                ) : (
                                    <></>
                                )}
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
    const { currentUser } = useCurrentUserCtx();
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
            <Grid md={9} xs={12}>
                <Stack spacing={3}>
                    {overview?.categories
                        .filter((c) => c.forums.length > 0)
                        .map((cat) => {
                            return (
                                <CategoryBlock
                                    category={cat}
                                    key={`category-${cat.forum_category_id}`}
                                />
                            );
                        })}
                </Stack>
            </Grid>
            <Grid md={3} xs={12}>
                <Stack spacing={3}>
                    <ContainerWithHeader
                        title={'Latest Activity'}
                        iconLeft={<TodayIcon />}
                    ></ContainerWithHeader>
                    {currentUser.permission_level >=
                        PermissionLevel.Moderator && (
                        <ContainerWithHeader
                            title={'Mod Tools'}
                            iconLeft={<ConstructionIcon />}
                        >
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
                    )}
                </Stack>
            </Grid>
        </Grid>
    );
};
