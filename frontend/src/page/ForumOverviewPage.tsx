import React, { JSX, useCallback, useEffect, useMemo, useState } from 'react';
import { Link as RouterLink } from 'react-router-dom';
import NiceModal from '@ebay/nice-modal-react';
import { Person2 } from '@mui/icons-material';
import AccessTimeIcon from '@mui/icons-material/AccessTime';
import CategoryIcon from '@mui/icons-material/Category';
import ChatIcon from '@mui/icons-material/Chat';
import ConstructionIcon from '@mui/icons-material/Construction';
import TodayIcon from '@mui/icons-material/Today';
import Avatar from '@mui/material/Avatar';
import Box from '@mui/material/Box';
import Button from '@mui/material/Button';
import Stack from '@mui/material/Stack';
import Tooltip from '@mui/material/Tooltip';
import Typography from '@mui/material/Typography';
import Grid from '@mui/material/Unstable_Grid2';
import { defaultAvatarHash, PermissionLevel } from '../api';
import {
    apiForumRecentActivity,
    apiGetForumOverview,
    Forum,
    ForumCategory,
    ForumMessage,
    ForumOverview
} from '../api/forum';
import { ContainerWithHeader } from '../component/ContainerWithHeader';
import { ContainerWithHeaderAndButtons } from '../component/ContainerWithHeaderAndButtons';
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
import { humanCount, renderDateTime, renderTime } from '../util/text';

const CategoryBlock = ({ category }: { category: ForumCategory }) => {
    const { currentUser } = useCurrentUserCtx();

    const onEdit = useCallback(async () => {
        try {
            await NiceModal.show(ModalForumCategoryEditor, {
                initial_forum_category_id: category.forum_category_id
            });
        } catch (e) {
            logErr(e);
        }
    }, [category.forum_category_id]);

    const buttons = useMemo(() => {
        return currentUser.permission_level >= PermissionLevel.Moderator
            ? [
                  <Button
                      size={'small'}
                      variant={'contained'}
                      color={'warning'}
                      key={`cat-edit-${category.forum_category_id}`}
                      startIcon={<ConstructionIcon />}
                      onClick={onEdit}
                  >
                      Edit
                  </Button>
              ]
            : [];
    }, [category.forum_category_id, currentUser.permission_level, onEdit]);

    return (
        <ContainerWithHeaderAndButtons
            title={category.title}
            iconLeft={<CategoryIcon />}
            buttons={buttons}
        >
            <Stack
                spacing={1}
                sx={{
                    overflow: 'hidden',
                    textOverflow: 'ellipsis',
                    whiteSpace: 'nowrap',
                    width: '100%'
                }}
            >
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
                                            <VCenterBox>
                                                <ForumRowLink
                                                    label={f.title}
                                                    to={`/forums/${f.forum_id}`}
                                                />
                                            </VCenterBox>
                                            <VCenterBox>
                                                <Typography variant={'body2'}>
                                                    {f.description}
                                                </Typography>
                                            </VCenterBox>
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
                                                variant={'body1'}
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
        </ContainerWithHeaderAndButtons>
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
                    <RecentMessageActivity />
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

export const RecentMessageActivity = () => {
    const [recent, setRecent] = useState<ForumMessage[]>([]);

    useEffect(() => {
        const abortController = new AbortController();
        apiForumRecentActivity()
            .then((act) => {
                setRecent(act);
            })
            .catch((e) => logErr(e));

        return () => abortController.abort();
    }, []);
    return (
        <ContainerWithHeader title={'Latest Activity'} iconLeft={<TodayIcon />}>
            <Stack spacing={1}>
                {recent.map((m) => {
                    return (
                        <Stack
                            direction={'row'}
                            key={`message-${m.forum_message_id}`}
                            spacing={1}
                            sx={{
                                overflow: 'hidden',
                                textOverflow: 'ellipsis',
                                whiteSpace: 'nowrap',
                                width: '100%'
                            }}
                        >
                            <VCenteredElement
                                icon={
                                    <Avatar
                                        alt={m.personaname}
                                        src={`https://avatars.akamai.steamstatic.com/${
                                            m.avatarhash ?? defaultAvatarHash
                                        }.jpg`}
                                    />
                                }
                            />
                            <Stack>
                                <Box
                                    sx={{
                                        overflow: 'hidden',
                                        textOverflow: 'ellipsis',
                                        whiteSpace: 'nowrap',
                                        width: '100%'
                                    }}
                                >
                                    <ForumRowLink
                                        variant={'body1'}
                                        label={m.title ?? ''}
                                        to={`/forums/thread/${m.forum_thread_id}`}
                                    />
                                </Box>
                                <Stack direction={'row'} spacing={1}>
                                    <AccessTimeIcon scale={0.5} />
                                    <VCenterBox>
                                        <Tooltip
                                            title={renderDateTime(m.created_on)}
                                        >
                                            <Typography variant={'body2'}>
                                                {renderTime(
                                                    m.created_on ?? new Date()
                                                )}
                                            </Typography>
                                        </Tooltip>
                                    </VCenterBox>
                                    <Person2 scale={0.5} />
                                    <VCenterBox>
                                        <Typography
                                            overflow={'hidden'}
                                            color={(theme) => {
                                                return theme.palette.text
                                                    .secondary;
                                            }}
                                            component={RouterLink}
                                            to={`/profile/${m.source_id}`}
                                            variant={'body2'}
                                        >
                                            {m.personaname}
                                        </Typography>
                                    </VCenterBox>
                                </Stack>
                            </Stack>
                        </Stack>
                    );
                })}
            </Stack>
        </ContainerWithHeader>
    );
};
