import React, { useEffect, useState } from 'react';
import Grid from '@mui/material/Grid';
import Paper from '@mui/material/Paper';
import { UserTable } from '../component/UserTable';
import { apiGetBans, Ban } from '../api';

export const Bans = (): JSX.Element => {
    const [bans, setBans] = useState<Ban[]>([]);

    useEffect(() => {
        apiGetBans().then((b) => {
            setBans(b);
        });
    }, []);

    return (
        <Grid container spacing={3} paddingTop={3}>
            <Grid item xs>
                <Paper elevation={1}>
                    <UserTable
                        rows={bans}
                        rowsPerPage={25}
                        defaultSortColumn={'ban_id'}
                        columns={[
                            {
                                tooltip: 'Bans',
                                label: 'Bans',
                                sortKey: 'ban_id'
                            }
                        ]}
                    />
                </Paper>
            </Grid>
        </Grid>
    );
};
