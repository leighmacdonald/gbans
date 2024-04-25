import { useCallback } from 'react';
import { Link as RouterLink } from 'react-router-dom';
import NiceModal from '@ebay/nice-modal-react';
import InsightsIcon from '@mui/icons-material/Insights';
import PublishIcon from '@mui/icons-material/Publish';
import Button from '@mui/material/Button';
import ButtonGroup from '@mui/material/ButtonGroup';
import Link from '@mui/material/Link';
import Grid from '@mui/material/Unstable_Grid2';
import { createLazyFileRoute } from '@tanstack/react-router';
import { format } from 'date-fns';
import formatDistanceToNowStrict from 'date-fns/formatDistanceToNowStrict';
import { isAfter } from 'date-fns/fp';
import { apiContests, Contest } from '../api';
import { ContainerWithHeader } from '../component/ContainerWithHeader';
import { ModalContestEntry } from '../component/modal';
import { LazyTableSimple } from '../component/table/LazyTableSimple';
import { logErr } from '../util/errors';

export const Route = createLazyFileRoute('/contests')({
    component: Contests
});

function Contests() {
    const onEnter = useCallback(async (contest_id: string) => {
        try {
            await NiceModal.show(ModalContestEntry, { contest_id });
        } catch (e) {
            logErr(e);
        }
    }, []);

    return (
        <ContainerWithHeader title={'Contests'} iconLeft={<InsightsIcon />}>
            <Grid container>
                <Grid xs={12}>
                    <LazyTableSimple<Contest>
                        fetchData={apiContests}
                        showPager={true}
                        defaultSortDir={'desc'}
                        defaultSortColumn={'date_end'}
                        columns={[
                            {
                                sortKey: 'title',
                                sortable: true,
                                label: 'Title',
                                tooltip: 'Title',
                                align: 'left',
                                renderer: (contest) => (
                                    <Link
                                        component={RouterLink}
                                        to={`/contests/${contest.contest_id}`}
                                        variant={'button'}
                                    >
                                        {contest.title}
                                    </Link>
                                )
                            },
                            {
                                sortKey: 'num_entries',
                                sortable: true,
                                label: 'Entries',
                                tooltip: 'Number of entries',
                                align: 'left'
                            },
                            {
                                sortKey: 'date_start',
                                sortable: true,
                                sortType: 'date',
                                label: 'Starts',
                                tooltip: 'Starting date',
                                align: 'left',
                                renderer: (contest) =>
                                    format(contest.date_start, 'dd/MM/yy H:m')
                            },
                            {
                                sortKey: 'date_end',
                                sortable: true,
                                sortType: 'date',
                                label: 'Ends',
                                tooltip: 'Ending date',
                                align: 'left',
                                renderer: (contest) =>
                                    format(contest.date_end, 'dd/MM/yy H:m')
                            },
                            {
                                sortable: true,
                                virtualKey: 'remaining',
                                virtual: true,
                                label: 'Remaining',
                                tooltip: 'Remaining Time',
                                align: 'left',
                                renderer: (contest) => {
                                    if (isAfter(contest.date_end, new Date())) {
                                        return 'Expired';
                                    }

                                    return formatDistanceToNowStrict(
                                        contest.date_end
                                    );
                                }
                            },
                            {
                                sortable: true,
                                virtualKey: 'actions',
                                virtual: true,
                                label: '',
                                tooltip: '',
                                align: 'center',
                                width: '200px',
                                renderer: (contest) => {
                                    return (
                                        <ButtonGroup
                                            sx={{
                                                marginTop: 1,
                                                marginBottom: 1
                                            }}
                                        >
                                            <Button
                                                fullWidth
                                                variant={'contained'}
                                                color={'success'}
                                                disabled={isAfter(
                                                    contest.date_end,
                                                    new Date()
                                                )}
                                                startIcon={<PublishIcon />}
                                                onClick={async () => {
                                                    await onEnter(
                                                        contest.contest_id
                                                    );
                                                }}
                                            >
                                                Submit Entry
                                            </Button>
                                        </ButtonGroup>
                                    );
                                }
                            }
                        ]}
                    />
                </Grid>
            </Grid>
        </ContainerWithHeader>
    );
}
