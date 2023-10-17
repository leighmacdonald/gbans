import React, { useCallback } from 'react';
import Grid from '@mui/material/Unstable_Grid2';
import { useContests } from '../api';
import { LoadingSpinner } from '../component/LoadingSpinner';
import Button from '@mui/material/Button';
import NiceModal from '@ebay/nice-modal-react';
import { ModalContestEditor } from '../component/modal';

export const AdminContests = () => {
    const { loading, contests } = useContests();

    const onNewContest = useCallback(async () => {
        await NiceModal.show(ModalContestEditor, {});
    }, []);

    return (
        <Grid container>
            <Grid xs={12}>
                <Button variant={'contained'} onClick={onNewContest}>
                    Create New Contest
                </Button>
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
