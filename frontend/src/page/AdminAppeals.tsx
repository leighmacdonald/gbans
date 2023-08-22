import Grid from '@mui/material/Unstable_Grid2';
import React, { useEffect, useMemo, useState } from 'react';
import Typography from '@mui/material/Typography';
import {
    apiGetAppeals,
    AppealOverview,
    AppealState,
    appealStateString,
    BanReason
} from '../api';
import { logErr } from '../util/errors';
import Paper from '@mui/material/Paper';
import format from 'date-fns/format';
import { Heading } from '../component/Heading';
import { steamIdQueryValue } from '../util/text';
import Button from '@mui/material/Button';
import { Link } from 'react-router-dom';
import { PersonCell } from '../component/PersonCell';
import { LazyTable } from '../component/LazyTable';
import { Order, RowsPerPage } from '../component/DataTable';
import { TablePagination } from '@mui/material';
import Box from '@mui/material/Box';
import FormControl from '@mui/material/FormControl';
import InputLabel from '@mui/material/InputLabel';
import Select, { SelectChangeEvent } from '@mui/material/Select';
import MenuItem from '@mui/material/MenuItem';
import CardContent from '@mui/material/CardContent';
import Card from '@mui/material/Card';
import CardActions from '@mui/material/CardActions';
import { noop } from 'lodash-es';
import { addDays, isAfter, isBefore } from 'date-fns/fp';

interface BasicStatCardProps {
    title: string;
    value: string | number;
    desc: string;
    actionLabel?: string;
    onAction?: () => void;
}

const BasicStatCard = ({
    title,
    value,
    desc,
    actionLabel,
    onAction
}: BasicStatCardProps) => (
    <Card sx={{ minWidth: 275 }} variant={'outlined'}>
        <CardContent>
            <Typography
                sx={{ fontSize: 14 }}
                color="text.secondary"
                gutterBottom
            >
                {title}
            </Typography>
            <Typography variant="h1" component="div">
                {value}
            </Typography>
            {/*<Typography sx={{ mb: 1.5 }} color="text.secondary">*/}
            {/*    adjective*/}
            {/*</Typography>*/}
            <Typography variant="body2">{desc}</Typography>
        </CardContent>
        {actionLabel && (
            <CardActions>
                <Button size="small" onClick={onAction ?? noop}>
                    {actionLabel}
                </Button>
            </CardActions>
        )}
    </Card>
);

export const AdminAppeals = () => {
    const [sortOrder, setSortOrder] = useState<Order>('desc');
    const [sortColumn, setSortColumn] =
        useState<keyof AppealOverview>('ban_id');
    const [appeals, setAppeals] = useState<AppealOverview[]>([]);
    const [rowPerPageCount, setRowPerPageCount] = useState<number>(
        RowsPerPage.Fifty
    );
    const [page, setPage] = useState(0);
    const [appealState, setAppealState] = useState<AppealState>(
        AppealState.Any
    );
    const [totalRows] = useState<number>(0);

    useEffect(() => {
        apiGetAppeals()
            .then((response) => {
                setAppeals(response.result || []);
            })
            .catch(logErr);
    }, []);

    const rows = useMemo(() => {
        if (appealState == AppealState.Any) {
            return appeals;
        }
        return appeals.filter((f) => f.appeal_state == appealState);
    }, [appealState, appeals]);

    const newAppeals = useMemo(() => {
        return appeals.filter(
            (value) =>
                value.appeal_state == AppealState.Open &&
                isAfter(addDays(-2, new Date()), value.updated_on)
        ).length;
    }, [appeals]);

    const deniedAppeals = useMemo(() => {
        return appeals.filter(
            (value) =>
                value.appeal_state == AppealState.Denied ||
                value.appeal_state == AppealState.NoAppeal
        ).length;
    }, [appeals]);

    const oldAppeals = useMemo(() => {
        return appeals.filter(
            (value) =>
                value.appeal_state == AppealState.Open &&
                isBefore(addDays(-2, new Date()), value.updated_on)
        ).length;
    }, [appeals]);

    const resolvedAppeals = useMemo(() => {
        return appeals.filter((value) => value.appeal_state != AppealState.Open)
            .length;
    }, [appeals]);

    const selectItems = useMemo(() => {
        return [
            AppealState.Any,
            AppealState.Open,
            AppealState.Denied,
            AppealState.Accepted,
            AppealState.Reduced,
            AppealState.NoAppeal
        ].map((as) => {
            return (
                <MenuItem value={as} key={`as-${as}`}>
                    {appealStateString(as)}
                </MenuItem>
            );
        });
    }, []);

    return (
        <Grid container spacing={3}>
            <Grid container spacing={2}>
                <Grid xs={6} md={3}>
                    <BasicStatCard
                        value={newAppeals}
                        title={'New/Open Appeals'}
                        desc={'Recently created & open appeals'}
                    />
                </Grid>

                <Grid xs={6} md={3}>
                    <BasicStatCard
                        value={oldAppeals}
                        title={'Old/Open'}
                        desc={
                            'Appeals with no activity for >2days, but not resolved'
                        }
                    />
                </Grid>

                <Grid xs={6} md={3}>
                    <BasicStatCard
                        value={deniedAppeals}
                        title={'Denied'}
                        desc={'Number of Denied & No Appeal '}
                    />
                </Grid>

                <Grid xs={6} md={3}>
                    <BasicStatCard
                        value={resolvedAppeals}
                        title={'Accepted'}
                        desc={'Users with accept appeals'}
                    />
                </Grid>
            </Grid>
            <Grid
                xs={12}
                container
                justifyContent="space-between"
                alignItems="center"
                flexDirection={{ xs: 'column', sm: 'row' }}
            >
                <Grid xs={3}>
                    <Box sx={{ width: 120 }}>
                        <FormControl fullWidth>
                            <InputLabel id="appeal-state-label">
                                Appeal State
                            </InputLabel>
                            <Select<AppealState>
                                labelId="appeal-state-label"
                                id="appeal-state"
                                label="Appeal State"
                                value={appealState ?? AppealState.Any}
                                onChange={(
                                    event: SelectChangeEvent<AppealState>
                                ) => {
                                    setAppealState(
                                        event.target.value as AppealState
                                    );
                                }}
                            >
                                {selectItems}
                            </Select>
                        </FormControl>
                    </Box>
                </Grid>
                <Grid xs={'auto'}>
                    <TablePagination
                        component="div"
                        variant={'head'}
                        page={page}
                        count={totalRows}
                        showFirstButton
                        showLastButton
                        rowsPerPage={rowPerPageCount}
                        onRowsPerPageChange={(
                            event: React.ChangeEvent<
                                HTMLInputElement | HTMLTextAreaElement
                            >
                        ) => {
                            setRowPerPageCount(
                                parseInt(event.target.value, 10)
                            );
                            setPage(0);
                        }}
                        onPageChange={(_, newPage) => {
                            setPage(newPage);
                        }}
                    />
                </Grid>
            </Grid>
            <Grid xs={12}>
                <Paper>
                    <Heading>Recent Open Appeal Activity</Heading>
                    <LazyTable<AppealOverview>
                        rows={rows}
                        sortOrder={sortOrder}
                        sortColumn={sortColumn}
                        onSortColumnChanged={async (column) => {
                            setSortColumn(column);
                        }}
                        onSortOrderChanged={async (direction) => {
                            setSortOrder(direction);
                        }}
                        columns={[
                            {
                                label: '#',
                                tooltip: 'Ban ID',
                                sortable: true,
                                align: 'left',
                                queryValue: (o) => `${o.ban_id}`,
                                renderer: (obj) => (
                                    <Button
                                        component={Link}
                                        variant={'contained'}
                                        to={`/ban/${obj.ban_id}`}
                                    >
                                        #{obj.ban_id}
                                    </Button>
                                )
                            },
                            {
                                label: 'Author',
                                tooltip: 'Author',
                                sortable: true,
                                align: 'left',
                                queryValue: (o) =>
                                    steamIdQueryValue(o.source_id),
                                renderer: (row) => (
                                    <PersonCell
                                        steam_id={row.source_id}
                                        personaname={
                                            row.source_persona_name ||
                                            row.source_id.toString()
                                        }
                                        avatar={row.source_avatar_full}
                                    ></PersonCell>
                                )
                            },
                            {
                                label: 'Target',
                                tooltip: 'Target',
                                sortable: true,
                                align: 'left',
                                queryValue: (o) =>
                                    steamIdQueryValue(o.target_id),
                                renderer: (row) => (
                                    <PersonCell
                                        steam_id={row.target_id}
                                        personaname={
                                            row.target_persona_name ||
                                            row.target_id.toString()
                                        }
                                        avatar={row.target_avatar_full}
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
                                sortable: true,
                                align: 'left',
                                width: '150px',
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
                                sortable: true,
                                align: 'left',
                                width: '150px',
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
                    />
                </Paper>
            </Grid>
        </Grid>
    );
};
