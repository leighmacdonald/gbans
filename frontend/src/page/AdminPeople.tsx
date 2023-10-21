import Paper from '@mui/material/Paper';
import Stack from '@mui/material/Stack';
import Typography from '@mui/material/Typography';
import Grid from '@mui/material/Unstable_Grid2';
import React, { useEffect, useState, JSX } from 'react';
import { apiGetPeople, Person } from '../api';
import { DataTable } from '../component/DataTable';
import { PersonCell } from '../component/PersonCell';

export const AdminPeople = (): JSX.Element => {
    const [people, setPeople] = useState<Person[]>([]);

    useEffect(() => {
        apiGetPeople().then((response) => {
            setPeople(response || []);
        });
    }, []);

    return (
        <Grid container spacing={2}>
            <Grid xs={12}>
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
                                            avatar_hash={row.avatar}
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
