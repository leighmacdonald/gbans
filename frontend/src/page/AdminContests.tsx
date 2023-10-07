import React from 'react';
import Grid from '@mui/material/Unstable_Grid2';
import { useContests } from '../api';
import { LoadingSpinner } from '../component/LoadingSpinner';
import Button from '@mui/material/Button';

export const AdminContests = () => {
    const { loading, contests } = useContests();

    return (
        <Grid container>
            <Grid xs={12}>
                <Button variant={'contained'}>Create</Button>
            </Grid>
            {loading ? (
                <Grid xs={12}>
                    <LoadingSpinner />
                </Grid>
            ) : (
                contests.map((contest) => {
                    return (
                        <Grid xs={12} key={`contest-${contest.contest_id}`}>
                            {contest.title}
                        </Grid>
                    );
                })
            )}
        </Grid>
    );
};
