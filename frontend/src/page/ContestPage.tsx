import React, { useCallback } from 'react';
import formatDistanceToNowStrict from 'date-fns/formatDistanceToNowStrict';
import Grid from '@mui/material/Unstable_Grid2';
import Typography from '@mui/material/Typography';
import EmojiEventsIcon from '@mui/icons-material/EmojiEvents';
import Stack from '@mui/material/Stack';
import Box from '@mui/material/Box';
import { isAfter } from 'date-fns/fp';
import { defaultAvatarHash, useContest, useContestEntries } from '../api';
import { ContainerWithHeader } from '../component/ContainerWithHeader';
import { useParams } from 'react-router-dom';
import { LoadingSpinner } from '../component/LoadingSpinner';
import { format } from 'date-fns';
import { LoadingPlaceholder } from '../component/LoadingPlaceholder';
import InfoIcon from '@mui/icons-material/Info';
import { PageNotFound } from './PageNotFound';
import PublishIcon from '@mui/icons-material/Publish';
import Button from '@mui/material/Button';
import NiceModal from '@ebay/nice-modal-react';
import { ModalContestEntry } from '../component/modal';
import { logErr } from '../util/errors';
import ThumbUpIcon from '@mui/icons-material/ThumbUp';
import ThumbDownIcon from '@mui/icons-material/ThumbDown';
import Paper from '@mui/material/Paper';
import Avatar from '@mui/material/Avatar';
import ButtonGroup from '@mui/material/ButtonGroup';
import { IconButton } from '@mui/material';

interface InfoBarProps {
    title: string;
    value: string | number;
    align?: 'left' | 'right';
}

export const InfoBar = ({ title, value, align = 'left' }: InfoBarProps) => {
    return (
        <Box>
            <Typography variant={'subtitle1'} fontWeight={500} align={align}>
                {title}
            </Typography>
            <Typography variant={'h3'} fontWeight={700} align={align}>
                {value}
            </Typography>
        </Box>
    );
};

export const ContestPage = () => {
    const { contest_id } = useParams();
    const { loading, contest } = useContest(contest_id);
    const { entries } = useContestEntries(contest_id ?? '');

    const onEnter = useCallback(async (contest_id: string) => {
        try {
            await NiceModal.show(ModalContestEntry, { contest_id });
        } catch (e) {
            logErr(e);
        }
    }, []);

    if (!contest_id) {
        return <PageNotFound />;
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
                                value={contest.num_entries}
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
                <Grid xs={12}>
                    <Stack spacing={2}>
                        {entries.map((value) => {
                            return (
                                <Stack key={value.contest_entry_id}>
                                    <Paper elevation={2}>
                                        <Stack direction={'row'}>
                                            <Avatar
                                                alt={value.personaname}
                                                src={`https://avatars.akamai.steamstatic.com/${defaultAvatarHash}.jpg`}
                                                variant={'square'}
                                                sx={{
                                                    height: '128px',
                                                    width: '128px'
                                                }}
                                            />

                                            <Grid container>
                                                <Grid xs={8}>
                                                    <Typography
                                                        variant={'body1'}
                                                    >
                                                        {value.description}
                                                    </Typography>
                                                </Grid>
                                            </Grid>
                                        </Stack>
                                    </Paper>
                                    <Stack direction={'row'}>
                                        <ButtonGroup fullWidth>
                                            <IconButton color={'success'}>
                                                <ThumbUpIcon />
                                            </IconButton>
                                            <IconButton color={'error'}>
                                                <ThumbDownIcon />
                                            </IconButton>
                                        </ButtonGroup>
                                    </Stack>
                                </Stack>
                            );
                        })}
                    </Stack>
                </Grid>
            </Grid>
        )
    );
};
