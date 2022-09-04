import Grid from '@mui/material/Grid';
import React, { useEffect, useState } from 'react';
import Typography from '@mui/material/Typography';
import { apiGetAppeals, BanReason, IAPIBanRecord } from '../api';
import { logErr } from '../util/errors';
import { DataTable } from '../component/DataTable';
import Paper from '@mui/material/Paper';
import format from 'date-fns/format';
import { Heading } from '../component/Heading';
import { steamIdQueryValue } from '../util/text';
import Button from '@mui/material/Button';
import { useNavigate } from 'react-router-dom';
import { PersonCell } from '../component/PersonCell';

export const AdminAppeals = (): JSX.Element => {
    const [appeals, setAppeals] = useState<IAPIBanRecord[]>([]);
    const navigate = useNavigate();

    useEffect(() => {
        apiGetAppeals()
            .then((response) => {
                setAppeals(response.result || []);
            })
            .catch(logErr);
    }, []);

    return (
        <Grid container spacing={3} paddingTop={3}>
            <Grid item xs={12}>
                <Paper>
                    <Heading>Recent Open Appeal Activity</Heading>
                    <DataTable
                        columns={[
                            {
                                label: '#',
                                tooltip: 'Ban ID',
                                sortKey: 'ban_id',
                                sortable: true,
                                align: 'left',
                                queryValue: (o) => `${o.ban_id}`,
                                renderer: (obj) => (
                                    <Button
                                        variant={'contained'}
                                        onClick={() => {
                                            navigate(`/ban/${obj.ban_id}`);
                                        }}
                                    >
                                        #{obj.ban_id}
                                    </Button>
                                )
                            },
                            {
                                label: 'Author',
                                tooltip: 'Author',
                                sortKey: 'source_id',
                                sortable: true,
                                align: 'left',
                                queryValue: (o) =>
                                    steamIdQueryValue(o.source_id),
                                renderer: (row) => (
                                    <PersonCell
                                        steam_id={row.source_id}
                                        personaname={row.source_id.toString()}
                                        avatar={''}
                                    ></PersonCell>
                                )
                            },
                            {
                                label: 'Target',
                                tooltip: 'Target',
                                sortKey: 'target_id',
                                sortable: true,
                                align: 'left',
                                queryValue: (o) =>
                                    steamIdQueryValue(o.target_id),
                                renderer: (row) => (
                                    <PersonCell
                                        steam_id={row.target_id}
                                        personaname={row.target_id.toString()}
                                        avatar={''}
                                    ></PersonCell>
                                )
                            },
                            {
                                label: 'Reason',
                                tooltip: 'Reason',
                                sortKey: 'reason',
                                sortable: true,
                                align: 'left',
                                queryValue: (o) => BanReason[o.reason],
                                renderer: (row) => (
                                    <Typography variant={'body1'}>
                                        {BanReason[row.reason]}
                                    </Typography>
                                )
                            },
                            {
                                label: 'Custom Reason',
                                tooltip: 'Custom',
                                sortKey: 'reason_text',
                                sortable: false,
                                align: 'left'
                            },
                            {
                                label: 'Created',
                                tooltip: 'Created On',
                                sortType: 'date',
                                align: 'left',
                                width: '150px',
                                virtual: true,
                                virtualKey: 'created_on',
                                renderer: (obj) => {
                                    return (
                                        <Typography variant={'body1'}>
                                            {format(
                                                obj.created_on,
                                                'yyyy-MM-dd'
                                            )}
                                        </Typography>
                                    );
                                }
                            },
                            {
                                label: 'Updated',
                                tooltip: 'Updated On',
                                sortType: 'date',
                                sortKey: 'updated_on',
                                align: 'left',
                                width: '150px',
                                virtual: false,
                                renderer: (obj) => {
                                    return (
                                        <Typography variant={'body1'}>
                                            {format(
                                                obj.created_on,
                                                'yyyy-MM-dd'
                                            )}
                                        </Typography>
                                    );
                                }
                            }
                        ]}
                        defaultSortColumn={'created_on'}
                        rowsPerPage={10}
                        rows={appeals}
                    />
                </Paper>
            </Grid>
        </Grid>
    );
};
