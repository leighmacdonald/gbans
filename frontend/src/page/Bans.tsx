import React, { useEffect, useState } from 'react';
import Grid from '@mui/material/Grid';
import Paper from '@mui/material/Paper';
import { DataTable } from '../component/DataTable';
import { apiGetBansSteam, IAPIBanRecord } from '../api';

export const Bans = (): JSX.Element => {
    const [bans, setBans] = useState<IAPIBanRecord[]>([]);

    useEffect(() => {
        apiGetBansSteam().then((b) => {
            setBans(b);
        });
    }, []);

    return (
        <Grid container spacing={3} paddingTop={3}>
            <Grid item xs>
                <Paper elevation={1}>
                    <DataTable
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
