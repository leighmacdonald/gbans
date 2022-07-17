import React, { useEffect, useState } from 'react';
import Grid from '@mui/material/Grid';
import Paper from '@mui/material/Paper';
import Typography from '@mui/material/Typography';
import Stack from '@mui/material/Stack';
import { apiGetPeople, Person } from '../api';
import { UserTable } from '../component/UserTable';

export const AdminPeople = (): JSX.Element => {
    const [people, setPeople] = useState<Person[]>([]);

    useEffect(() => {
        apiGetPeople().then((p) => {
            setPeople(p);
        });
    }, []);

    return (
        <Grid container spacing={3} paddingTop={3}>
            <Grid item xs={12}>
                <Paper elevation={1}>
                    <Stack padding={3}>
                        <Typography variant={'h2'}>Known Players</Typography>
                        <UserTable
                            columns={[
                                {
                                    label: 'Steam ID',
                                    tooltip: 'Steam ID',
                                    sortKey: 'steam_id'
                                }
                            ]}
                            defaultSortColumn={'updated_on'}
                            rowsPerPage={100}
                            rows={people}
                        />
                    </Stack>
                </Paper>
            </Grid>
        </Grid>
    );
};
