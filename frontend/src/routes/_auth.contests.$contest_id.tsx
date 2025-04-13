import { useCallback, useEffect, useMemo, useState } from 'react';
import NiceModal from '@ebay/nice-modal-react';
import EmojiEventsIcon from '@mui/icons-material/EmojiEvents';
import InfoIcon from '@mui/icons-material/Info';
import PageviewIcon from '@mui/icons-material/Pageview';
import PublishIcon from '@mui/icons-material/Publish';
import ThumbDownIcon from '@mui/icons-material/ThumbDown';
import ThumbUpIcon from '@mui/icons-material/ThumbUp';
import Button from '@mui/material/Button';
import ButtonGroup from '@mui/material/ButtonGroup';
import Grid from '@mui/material/Grid';
import Paper from '@mui/material/Paper';
import Stack from '@mui/material/Stack';
import Typography from '@mui/material/Typography';
import { useQuery } from '@tanstack/react-query';
import { createFileRoute, useRouteContext } from '@tanstack/react-router';
import { format } from 'date-fns';
import formatDistanceToNowStrict from 'date-fns/formatDistanceToNowStrict';
import { isAfter } from 'date-fns/fp';
import { apiContest, apiContestEntries, apiContestEntryVote, ContestEntry, PermissionLevel } from '../api';
import { Asset, mediaType, MediaTypes } from '../api/media.ts';
import { ContainerWithHeader } from '../component/ContainerWithHeader.tsx';
import { InfoBar } from '../component/InfoBar.tsx';
import { LoadingPlaceholder } from '../component/LoadingPlaceholder.tsx';
import { LoadingSpinner } from '../component/LoadingSpinner.tsx';
import { MarkDownRenderer } from '../component/MarkdownRenderer.tsx';
import { PersonCell } from '../component/PersonCell.tsx';
import { Title } from '../component/Title';
import { VCenterBox } from '../component/VCenterBox.tsx';
import { ModalAssetViewer, ModalContestEntry, ModalContestEntryDelete } from '../component/modal';
import { useUserFlashCtx } from '../hooks/useUserFlashCtx.ts';
import { logErr } from '../util/errors.ts';
import { ensureFeatureEnabled } from '../util/features.ts';
import { humanFileSize } from '../util/text.tsx';
import { PageNotFound } from './_auth.page-not-found.tsx';

export const Route = createFileRoute('/_auth/contests/$contest_id')({
    component: Contest,
    beforeLoad: () => {
        ensureFeatureEnabled('contests_enabled');
    }
});

function Contest() {
    const { contest_id } = Route.useParams();
    const [entries, setEntries] = useState<ContestEntry[]>([]);
    const [entriesLoading, setEntriesLoading] = useState(false);
    const { hasPermission, profile } = useRouteContext({ from: '/_auth/contests/$contest_id' });
    const { sendFlash } = useUserFlashCtx();

    const {
        data: contest,
        isLoading,
        isError
    } = useQuery({
        queryKey: ['contest', { contest_id }],
        queryFn: async () => {
            return await apiContest(Number(contest_id));
        }
    });

    const onEnter = useCallback(async (contest_id: string) => {
        try {
            await NiceModal.show(ModalContestEntry, { contest_id });
        } catch (e) {
            logErr(e);
        }
    }, []);

    const updateEntries = useCallback(() => {
        if (!contest?.contest_id) {
            return;
        }
        setEntriesLoading(true);
        apiContestEntries(contest?.contest_id)
            .then((entries) => {
                setEntries(entries);
            })
            .catch(logErr)
            .finally(() => {
                setEntriesLoading(false);
            });
    }, [contest?.contest_id]);

    useEffect(() => {
        updateEntries();
    }, [contest?.contest_id, updateEntries]);

    const showEntries = useMemo(() => {
        return (contest && !contest.hide_submissions) || hasPermission(PermissionLevel.Moderator);
    }, [contest, hasPermission]);

    const vote = useCallback(
        async (contest_entry_id: string, up_vote: boolean) => {
            if (!contest?.contest_id) {
                return;
            }
            try {
                await apiContestEntryVote(contest?.contest_id, contest_entry_id, up_vote);
                updateEntries();
            } catch (e) {
                logErr(e);
            }
        },
        [contest?.contest_id, updateEntries]
    );

    const onViewAsset = useCallback(async (asset: Asset) => {
        await NiceModal.show(ModalAssetViewer, asset);
    }, []);

    const onDeleteEntry = useCallback(
        async (contest_entry_id: string) => {
            try {
                await NiceModal.show(ModalContestEntryDelete, {
                    contest_entry_id
                });
                setEntries((prevState) => {
                    return prevState.filter((v) => v.contest_entry_id != contest_entry_id);
                });
                sendFlash('success', `Entry deleted successfully`);
            } catch (e) {
                sendFlash('error', `Failed to delete entry: ${e}`);
            }
        },
        [sendFlash]
    );

    if (!contest_id) {
        return <PageNotFound />;
    }
    if (isError) {
        return <PageNotFound />;
    }
    return isLoading ? (
        <LoadingPlaceholder />
    ) : (
        contest && (
            <Grid container spacing={3}>
                <Title>{contest?.title ?? 'Contest'}</Title>
                <Grid size={{ xs: 8 }}>
                    <ContainerWithHeader
                        title={`Contest: ${contest?.title}`}
                        iconLeft={isLoading ? <LoadingSpinner /> : <EmojiEventsIcon />}
                    >
                        {isLoading ? (
                            <LoadingSpinner />
                        ) : (
                            contest && (
                                <Grid container>
                                    <Grid size={{ xs: 12 }} minHeight={400}>
                                        <Typography variant={'body1'} padding={2}>
                                            {contest?.description}
                                        </Typography>
                                    </Grid>
                                </Grid>
                            )
                        )}
                    </ContainerWithHeader>
                </Grid>
                <Grid size={{ xs: 4 }}>
                    <ContainerWithHeader
                        title={`Contest Details`}
                        iconLeft={isLoading ? <LoadingSpinner /> : <InfoIcon />}
                    >
                        <Stack spacing={2}>
                            <InfoBar
                                title={'Starting Date'}
                                value={format(contest.date_start, 'dd/MM/yy H:m')}
                                align={'right'}
                            />

                            <InfoBar
                                title={'Ending Date'}
                                value={format(contest.date_end, 'dd/MM/yy H:m')}
                                align={'right'}
                            />

                            <InfoBar
                                title={'Remaining'}
                                value={
                                    isAfter(contest.date_end, new Date())
                                        ? 'Expired'
                                        : formatDistanceToNowStrict(contest.date_end)
                                }
                                align={'right'}
                            />

                            <InfoBar title={'Max Entries Per User'} value={contest.max_submissions} align={'right'} />

                            <InfoBar title={'Total Entries'} value={entries.length} align={'right'} />
                            <Button
                                fullWidth
                                variant={'contained'}
                                color={'success'}
                                disabled={isAfter(contest.date_end, new Date())}
                                startIcon={<PublishIcon />}
                                onClick={async () => {
                                    await onEnter(contest.contest_id as string);
                                }}
                            >
                                Submit Entry
                            </Button>
                        </Stack>
                    </ContainerWithHeader>
                </Grid>
                {entriesLoading ? (
                    <LoadingSpinner />
                ) : (
                    <>
                        {!showEntries && (
                            <Grid size={{ xs: 12 }}>
                                <Paper>
                                    <Typography variant={'subtitle1'} align={'center'} padding={4}>
                                        Entries from other contestants are hidden.
                                    </Typography>
                                </Paper>
                            </Grid>
                        )}
                        <Grid size={{ xs: 12 }}>
                            <Stack spacing={2}>
                                {entries
                                    .filter((e) => showEntries || e.steam_id == profile.steam_id)
                                    .map((entry) => {
                                        return (
                                            <Stack key={entry.contest_entry_id}>
                                                <Paper elevation={2}>
                                                    <Grid container>
                                                        <Grid size={{ xs: 8 }} padding={2}>
                                                            <Typography variant={'subtitle1'}>Description</Typography>
                                                            <MarkDownRenderer
                                                                body_md={
                                                                    entry.description != ''
                                                                        ? entry.description
                                                                        : 'No description provided'
                                                                }
                                                            />
                                                        </Grid>
                                                        <Grid size={{ xs: 4 }} padding={2}>
                                                            <PersonCell
                                                                steam_id={entry.steam_id}
                                                                personaname={entry.personaname}
                                                                avatar_hash={entry.avatar_hash}
                                                            />
                                                            <Typography variant={'subtitle1'}>File Details</Typography>
                                                            <Typography variant={'body2'}>
                                                                {entry.asset.name}
                                                            </Typography>
                                                            <Typography variant={'body2'}>
                                                                {entry.asset.mime_type}
                                                            </Typography>
                                                            <Typography variant={'body2'}>
                                                                {humanFileSize(entry.asset.size)}
                                                            </Typography>
                                                            <ButtonGroup fullWidth>
                                                                <Button
                                                                    disabled={
                                                                        !(
                                                                            hasPermission(PermissionLevel.Moderator) ||
                                                                            profile.steam_id == entry.steam_id
                                                                        )
                                                                    }
                                                                    color={'error'}
                                                                    variant={'contained'}
                                                                    onClick={async () => {
                                                                        await onDeleteEntry(entry.contest_entry_id);
                                                                    }}
                                                                >
                                                                    Delete
                                                                </Button>

                                                                {mediaType(entry.asset.mime_type) !=
                                                                MediaTypes.other ? (
                                                                    <Button
                                                                        startIcon={<PageviewIcon />}
                                                                        fullWidth
                                                                        variant={'contained'}
                                                                        color={'success'}
                                                                        onClick={async () => {
                                                                            await onViewAsset(entry.asset);
                                                                        }}
                                                                    >
                                                                        View
                                                                    </Button>
                                                                ) : (
                                                                    <Button>Download</Button>
                                                                )}
                                                            </ButtonGroup>
                                                        </Grid>
                                                    </Grid>
                                                </Paper>
                                                <Stack direction={'row'} padding={1} spacing={2}>
                                                    <ButtonGroup
                                                        disabled={
                                                            !contest.voting || isAfter(contest.date_end, new Date())
                                                        }
                                                    >
                                                        <Button
                                                            size={'small'}
                                                            variant={'contained'}
                                                            startIcon={<ThumbUpIcon />}
                                                            color={'success'}
                                                            onClick={async () => {
                                                                await vote(entry.contest_entry_id, true);
                                                            }}
                                                        >
                                                            {entry.votes_up}
                                                        </Button>
                                                        <Button
                                                            size={'small'}
                                                            variant={'contained'}
                                                            startIcon={<ThumbDownIcon />}
                                                            color={'error'}
                                                            disabled={!contest.down_votes}
                                                            onClick={async () => {
                                                                await vote(entry.contest_entry_id, false);
                                                            }}
                                                        >
                                                            {entry.votes_down}
                                                        </Button>
                                                    </ButtonGroup>
                                                    <VCenterBox>
                                                        <Typography variant={'caption'}>
                                                            {`Updated: ${format(entry.updated_on, 'dd/MM/yy H:m')}`}
                                                        </Typography>
                                                    </VCenterBox>
                                                </Stack>
                                            </Stack>
                                        );
                                    })}
                            </Stack>
                        </Grid>
                    </>
                )}
            </Grid>
        )
    );
}
