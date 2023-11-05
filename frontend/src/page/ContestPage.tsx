import React, { useCallback, useEffect, useMemo, useState } from 'react';
import { useParams } from 'react-router-dom';
import NiceModal from '@ebay/nice-modal-react';
import EmojiEventsIcon from '@mui/icons-material/EmojiEvents';
import InfoIcon from '@mui/icons-material/Info';
import PageviewIcon from '@mui/icons-material/Pageview';
import PublishIcon from '@mui/icons-material/Publish';
import ThumbDownIcon from '@mui/icons-material/ThumbDown';
import ThumbUpIcon from '@mui/icons-material/ThumbUp';
import Button from '@mui/material/Button';
import ButtonGroup from '@mui/material/ButtonGroup';
import Paper from '@mui/material/Paper';
import Stack from '@mui/material/Stack';
import Typography from '@mui/material/Typography';
import Grid from '@mui/material/Unstable_Grid2';
import { format } from 'date-fns';
import formatDistanceToNowStrict from 'date-fns/formatDistanceToNowStrict';
import { isAfter } from 'date-fns/fp';
import {
    apiContestEntries,
    apiContestEntryVote,
    ContestEntry,
    ErrorCode,
    PermissionLevel,
    useContest
} from '../api';
import { Asset } from '../api/media';
import { ContainerWithHeader } from '../component/ContainerWithHeader';
import { InfoBar } from '../component/InfoBar';
import { LoadingPlaceholder } from '../component/LoadingPlaceholder';
import { LoadingSpinner } from '../component/LoadingSpinner';
import { PersonCell } from '../component/PersonCell';
import { VCenterBox } from '../component/VCenterBox';
import {
    ModalAssetViewer,
    ModalContestEntry,
    ModalContestEntryDelete
} from '../component/modal';
import { mediaType, MediaTypes } from '../component/modal/AssetViewer';
import { useCurrentUserCtx } from '../contexts/CurrentUserCtx';
import { useUserFlashCtx } from '../contexts/UserFlashCtx';
import { logErr } from '../util/errors';
import { humanFileSize } from '../util/text';
import { PageNotFound } from './PageNotFound';

export const ContestPage = () => {
    const { contest_id } = useParams();
    const { loading, contest, error } = useContest(contest_id);
    const [entries, setEntries] = useState<ContestEntry[]>([]);
    const [entriesLoading, setEntriesLoading] = useState(false);
    const { currentUser } = useCurrentUserCtx();
    const { sendFlash } = useUserFlashCtx();

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
        return (
            (contest && !contest.hide_submissions) ||
            currentUser.permission_level >= PermissionLevel.Moderator
        );
    }, [contest, currentUser.permission_level]);

    const vote = useCallback(
        async (contest_entry_id: string, up_vote: boolean) => {
            if (!contest?.contest_id) {
                return;
            }
            try {
                await apiContestEntryVote(
                    contest?.contest_id,
                    contest_entry_id,
                    up_vote
                );
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
                    return prevState.filter(
                        (v) => v.contest_entry_id != contest_entry_id
                    );
                });
                sendFlash('success', `Entry deleted successfully`);
            } catch (e) {
                sendFlash('error', `Failed to delete entry: ${e}`);
            }
        },
        [sendFlash]
    );

    if (!contest_id) {
        return <PageNotFound error={'Invalid Contest ID'} />;
    }
    if (error && error.code == ErrorCode.PermissionDenied) {
        return (
            <PageNotFound
                heading={'Cannot Load Contest'}
                error={error.message}
            />
        );
    }
    return loading ? (
        <LoadingPlaceholder />
    ) : (
        contest && (
            <Grid container spacing={3}>
                <Grid xs={8}>
                    <ContainerWithHeader
                        title={`Contest: ${contest?.title}`}
                        iconLeft={
                            loading ? <LoadingSpinner /> : <EmojiEventsIcon />
                        }
                    >
                        {loading ? (
                            <LoadingSpinner />
                        ) : (
                            contest && (
                                <Grid container>
                                    <Grid xs={12} minHeight={400}>
                                        <Typography
                                            variant={'body1'}
                                            padding={2}
                                        >
                                            {contest?.description}
                                        </Typography>
                                    </Grid>
                                </Grid>
                            )
                        )}
                    </ContainerWithHeader>
                </Grid>
                <Grid xs={4}>
                    <ContainerWithHeader
                        title={`Contest Details`}
                        iconLeft={loading ? <LoadingSpinner /> : <InfoIcon />}
                    >
                        <Stack spacing={2}>
                            <InfoBar
                                title={'Starting Date'}
                                value={format(
                                    contest.date_start,
                                    'dd/MM/yy H:m'
                                )}
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
                                        : formatDistanceToNowStrict(
                                              contest.date_end
                                          )
                                }
                                align={'right'}
                            />

                            <InfoBar
                                title={'Max Entries Per User'}
                                value={contest.max_submissions}
                                align={'right'}
                            />

                            <InfoBar
                                title={'Total Entries'}
                                value={entries.length}
                                align={'right'}
                            />
                            <Button
                                fullWidth
                                variant={'contained'}
                                color={'success'}
                                disabled={isAfter(contest.date_end, new Date())}
                                startIcon={<PublishIcon />}
                                onClick={async () => {
                                    await onEnter(contest.contest_id);
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
                            <Grid xs={12}>
                                <Paper>
                                    <Typography
                                        variant={'subtitle1'}
                                        align={'center'}
                                        padding={4}
                                    >
                                        Entries from other contestants are
                                        hidden.
                                    </Typography>
                                </Paper>
                            </Grid>
                        )}
                        <Grid xs={12}>
                            <Stack spacing={2}>
                                {entries
                                    .filter(
                                        (e) =>
                                            showEntries ||
                                            e.steam_id == currentUser.steam_id
                                    )
                                    .map((entry) => {
                                        return (
                                            <Stack key={entry.contest_entry_id}>
                                                <Paper elevation={2}>
                                                    <Grid container>
                                                        <Grid
                                                            xs={8}
                                                            padding={2}
                                                        >
                                                            <Typography
                                                                variant={
                                                                    'subtitle1'
                                                                }
                                                            >
                                                                Description
                                                            </Typography>
                                                            <Typography
                                                                variant={
                                                                    'body1'
                                                                }
                                                            >
                                                                {entry.description !=
                                                                ''
                                                                    ? entry.description
                                                                    : 'No description provided'}
                                                            </Typography>
                                                        </Grid>
                                                        <Grid
                                                            xs={4}
                                                            padding={2}
                                                        >
                                                            <PersonCell
                                                                steam_id={
                                                                    entry.steam_id
                                                                }
                                                                personaname={
                                                                    entry.personaname
                                                                }
                                                                avatar_hash={
                                                                    entry.avatar_hash
                                                                }
                                                            />
                                                            <Typography
                                                                variant={
                                                                    'subtitle1'
                                                                }
                                                            >
                                                                File Details
                                                            </Typography>
                                                            <Typography
                                                                variant={
                                                                    'body2'
                                                                }
                                                            >
                                                                {
                                                                    entry.asset
                                                                        .name
                                                                }
                                                            </Typography>
                                                            <Typography
                                                                variant={
                                                                    'body2'
                                                                }
                                                            >
                                                                {
                                                                    entry.asset
                                                                        .mime_type
                                                                }
                                                            </Typography>
                                                            <Typography
                                                                variant={
                                                                    'body2'
                                                                }
                                                            >
                                                                {humanFileSize(
                                                                    entry.asset
                                                                        .size
                                                                )}
                                                            </Typography>
                                                            <ButtonGroup
                                                                fullWidth
                                                            >
                                                                <Button
                                                                    disabled={
                                                                        !(
                                                                            currentUser.permission_level >=
                                                                                PermissionLevel.Moderator ||
                                                                            currentUser.steam_id ==
                                                                                entry.steam_id
                                                                        )
                                                                    }
                                                                    color={
                                                                        'error'
                                                                    }
                                                                    variant={
                                                                        'contained'
                                                                    }
                                                                    onClick={async () => {
                                                                        await onDeleteEntry(
                                                                            entry.contest_entry_id
                                                                        );
                                                                    }}
                                                                >
                                                                    Delete
                                                                </Button>

                                                                {mediaType(
                                                                    entry.asset
                                                                        .mime_type
                                                                ) !=
                                                                MediaTypes.other ? (
                                                                    <Button
                                                                        startIcon={
                                                                            <PageviewIcon />
                                                                        }
                                                                        fullWidth
                                                                        variant={
                                                                            'contained'
                                                                        }
                                                                        color={
                                                                            'success'
                                                                        }
                                                                        onClick={async () => {
                                                                            await onViewAsset(
                                                                                entry.asset
                                                                            );
                                                                        }}
                                                                    >
                                                                        View
                                                                    </Button>
                                                                ) : (
                                                                    <Button>
                                                                        Download
                                                                    </Button>
                                                                )}
                                                            </ButtonGroup>
                                                        </Grid>
                                                    </Grid>
                                                </Paper>
                                                <Stack
                                                    direction={'row'}
                                                    padding={1}
                                                    spacing={2}
                                                >
                                                    <ButtonGroup
                                                        disabled={
                                                            !contest.voting ||
                                                            isAfter(
                                                                contest.date_end,
                                                                new Date()
                                                            )
                                                        }
                                                    >
                                                        <Button
                                                            size={'small'}
                                                            variant={
                                                                'contained'
                                                            }
                                                            startIcon={
                                                                <ThumbUpIcon />
                                                            }
                                                            color={'success'}
                                                            onClick={async () => {
                                                                await vote(
                                                                    entry.contest_entry_id,
                                                                    true
                                                                );
                                                            }}
                                                        >
                                                            {entry.votes_up}
                                                        </Button>
                                                        <Button
                                                            size={'small'}
                                                            variant={
                                                                'contained'
                                                            }
                                                            startIcon={
                                                                <ThumbDownIcon />
                                                            }
                                                            color={'error'}
                                                            disabled={
                                                                !contest.down_votes
                                                            }
                                                            onClick={async () => {
                                                                await vote(
                                                                    entry.contest_entry_id,
                                                                    false
                                                                );
                                                            }}
                                                        >
                                                            {entry.votes_down}
                                                        </Button>
                                                    </ButtonGroup>
                                                    <VCenterBox>
                                                        <Typography
                                                            variant={'caption'}
                                                        >
                                                            {`Updated: ${format(
                                                                entry.updated_on,
                                                                'dd/MM/yy H:m'
                                                            )}`}
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
};
