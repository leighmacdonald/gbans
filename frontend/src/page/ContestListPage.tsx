import React from 'react';
import Grid from '@mui/material/Unstable_Grid2';
import { apiContests, Contest } from '../api';
import InsightsIcon from '@mui/icons-material/Insights';
import { LazyTableSimple } from '../component/LazyTableSimple';
import { ContainerWithHeader } from '../component/ContainerWithHeader';

export const ContestListPage = () => {
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
                                tooltip: 'Title'
                            }
                        ]}
                        defaultSortColumn={'date_start'}
                    />
                </Grid>
            </Grid>
        </ContainerWithHeader>
    );
};
