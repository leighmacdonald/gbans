import React, { useEffect, useState } from 'react';
import Grid from '@mui/material/Grid';
import Paper from '@mui/material/Paper';
import Typography from '@mui/material/Typography';
import Stack from '@mui/material/Stack';
import { apiGetPeople, Person } from '../api';
import { DataTable } from '../component/DataTable';
import { PersonCell } from '../component/PersonCell';

export const AdminPeople = (): JSX.Element => {
    const [people, setPeople] = useState<Person[]>([]);

    useEffect(() => {
        apiGetPeople().then((response) => {
            setPeople(response.result || []);
        });
    }, []);

    return (
        <Grid container spacing={3} paddingTop={3}>
            <Grid item xs={12}>
                <Paper elevation={1}>
                    <Stack padding={3}>
                        <Typography variant={'h2'}>Known Players</Typography>
                        <DataTable
                            columns={[
                                {
                                    label: 'Steam ID',
                                    tooltip: 'Steam ID',
                                    sortKey: 'steam_id',
                                    queryValue: (o) => `${o.personaname}`,
                                    renderer: (row) => (
                                        <PersonCell
                                            steam_id={row.steam_id}
                                            personaname={row.personaname}
                                            avatar={row.avatar}
                                        />
                                    )
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
