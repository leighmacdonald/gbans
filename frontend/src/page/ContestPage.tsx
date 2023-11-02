import React, { useCallback, useEffect, useMemo, useState } from 'react';
import { useParams } from 'react-router-dom';
import NiceModal from '@ebay/nice-modal-react';
import EmojiEventsIcon from '@mui/icons-material/EmojiEvents';
import InfoIcon from '@mui/icons-material/Info';
import PublishIcon from '@mui/icons-material/Publish';
import ThumbDownIcon from '@mui/icons-material/ThumbDown';
import ThumbUpIcon from '@mui/icons-material/ThumbUp';
import Avatar from '@mui/material/Avatar';
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
    defaultAvatarHash,
    useContest
} from '../api';
import { ContainerWithHeader } from '../component/ContainerWithHeader';
import { InfoBar } from '../component/InfoBar';
import { LoadingPlaceholder } from '../component/LoadingPlaceholder';
import { LoadingSpinner } from '../component/LoadingSpinner';
import { ModalContestEntry } from '../component/modal';
import { logErr } from '../util/errors';
import { PageNotFound } from './PageNotFound';

export const ContestPage = () => {
    const { contest_id } = useParams();
    const { loading, contest } = useContest(contest_id);
    const [entries, setEntries] = useState<ContestEntry[]>([]);
    const [entriesLoading, setEntriesLoading] = useState(false);

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
            contest && contest.hide_submissions && !isAfter(contest.date_end)
        );
    }, [contest]);

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

    if (!contest_id) {
        return <PageNotFound error={'Invalid Contest ID'} />;
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
                                    <Grid xs={12}>
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
                        align={'flex-end'}
                        title={`Contest Details`}
                        iconRight={loading ? <LoadingSpinner /> : <InfoIcon />}
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
                                    isAfter(contest.date_end)
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
                {showEntries ? (
                    <Grid xs={12}>
                        <Paper>
                            <Typography
                                variant={'subtitle1'}
                                align={'center'}
                                padding={4}
                            >
                                Entries are hidden until contest has expired.
                            </Typography>
                        </Paper>
                    </Grid>
                ) : entriesLoading ? (
                    <LoadingSpinner />
                ) : (
                    <Grid xs={12}>
                        <Stack spacing={2}>
                            {entries.map((entry) => {
                                return (
                                    <Stack key={entry.contest_entry_id}>
                                        <Paper elevation={2}>
                                            <Stack direction={'row'}>
                                                <Avatar
                                                    alt={entry.personaname}
                                                    src={`https://avatars.akamai.steamstatic.com/${defaultAvatarHash}.jpg`}
                                                    variant={'square'}
                                                    sx={{
                                                        height: '128px',
                                                        width: '128px',
                                                        padding: 2
                                                    }}
                                                />

                                                <Grid container>
                                                    <Grid xs={8} padding={2}>
                                                        <Typography
                                                            variant={'body1'}
                                                        >
                                                            {entry.description}
                                                        </Typography>
                                                    </Grid>
                                                </Grid>
                                            </Stack>
                                        </Paper>
                                        <Stack direction={'row'} padding={1}>
                                            <ButtonGroup
                                                disabled={!contest.voting}
                                            >
                                                <Button
                                                    size={'small'}
                                                    variant={'contained'}
                                                    startIcon={<ThumbUpIcon />}
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
                                                    variant={'contained'}
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
                                        </Stack>
                                    </Stack>
                                );
                            })}
                        </Stack>
                    </Grid>
                )}
            </Grid>
        )
    );
};
