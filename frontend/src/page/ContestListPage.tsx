import React, { useCallback } from 'react';
import Grid from '@mui/material/Unstable_Grid2';
import { apiContests, Contest } from '../api';
import InsightsIcon from '@mui/icons-material/Insights';
import { LazyTableSimple } from '../component/LazyTableSimple';
import { ContainerWithHeader } from '../component/ContainerWithHeader';
import { format, formatDistanceToNow } from 'date-fns';
import ButtonGroup from '@mui/material/ButtonGroup';
import Button from '@mui/material/Button';
import { logErr } from '../util/errors';
import NiceModal from '@ebay/nice-modal-react';
import { ModalContestEntry } from '../component/modal';

export const ContestListPage = () => {
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
                        columns={[
                            {
                                sortKey: 'title',
                                sortable: true,
                                label: 'Title',
                                tooltip: 'Title',
                                align: 'left'
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
                                renderer: (obj) =>
                                    format(obj.date_start, 'H:m dd/MM/yyyy')
                            },
                            {
                                sortKey: 'date_end',
                                sortable: true,
                                sortType: 'date',
                                label: 'Ends',
                                tooltip: 'Ending date',
                                align: 'left',
                                renderer: (obj) =>
                                    format(obj.date_end, 'H:m dd/MM/yyyy')
                            },
                            {
                                sortable: true,
                                virtualKey: 'remaining',
                                virtual: true,
                                label: 'Remaining',
                                tooltip: 'Remaining Time',
                                align: 'left',
                                renderer: (obj) =>
                                    formatDistanceToNow(obj.date_end)
                            },
                            {
                                sortable: true,
                                virtualKey: 'actions',
                                virtual: true,
                                label: '',
                                tooltip: '',
                                align: 'center',
                                width: '200px',
                                renderer: (obj) => {
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
                                                onClick={async () => {
                                                    await onEnter(
                                                        obj.contest_id
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
                        defaultSortColumn={'date_start'}
                    />
                </Grid>
            </Grid>
        </ContainerWithHeader>
    );
};
