import React, { ReactNode, useEffect, useMemo, useState } from 'react';
import { Link } from 'react-router-dom';
import DoNotDisturbIcon from '@mui/icons-material/DoNotDisturb';
import FiberNewIcon from '@mui/icons-material/FiberNew';
import GppGoodIcon from '@mui/icons-material/GppGood';
import SnoozeIcon from '@mui/icons-material/Snooze';
import { TablePagination } from '@mui/material';
import Box from '@mui/material/Box';
import Button from '@mui/material/Button';
import Card from '@mui/material/Card';
import CardActions from '@mui/material/CardActions';
import CardContent from '@mui/material/CardContent';
import FormControl from '@mui/material/FormControl';
import InputLabel from '@mui/material/InputLabel';
import MenuItem from '@mui/material/MenuItem';
import Select, { SelectChangeEvent } from '@mui/material/Select';
import Stack from '@mui/material/Stack';
import Typography from '@mui/material/Typography';
import Grid from '@mui/material/Unstable_Grid2';
import format from 'date-fns/format';
import { addDays, isAfter, isBefore } from 'date-fns/fp';
import { noop } from 'lodash-es';
import {
    apiGetAppeals,
    AppealOverview,
    AppealState,
    appealStateString,
    BanReason
} from '../api';
import { ContainerWithHeader } from '../component/ContainerWithHeader';
import { Order, RowsPerPage } from '../component/DataTable';
import { LazyTable } from '../component/LazyTable';
import { LazyTablePaginator } from '../component/LazyTablePaginator';
import { LoadingSpinner } from '../component/LoadingSpinner';
import { PersonCell } from '../component/PersonCell';
import { logErr } from '../util/errors';
import { steamIdQueryValue } from '../util/text';

interface BasicStatCardProps {
    title: string;
    value: string | number;
    desc: string;
    actionLabel?: string;
    onAction?: () => void;
    icon?: ReactNode;
}

const BasicStatCard = ({
    title,
    value,
    desc,
    actionLabel,
    onAction,
    icon
}: BasicStatCardProps) => (
    <Card sx={{ minWidth: 275 }} variant={'outlined'}>
        <CardContent>
            <Stack direction={'row'} spacing={1}>
                {icon}
                <Typography
                    sx={{ fontSize: 14 }}
                    color="text.secondary"
                    gutterBottom
                >
                    {title}
                </Typography>
            </Stack>
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
    const [rowPerPageCount, setRowPerPageCount] = useState<number>(
        RowsPerPage.Fifty
    );
    const [page, setPage] = useState(0);
    const [appeals, setAppeals] = useState<AppealOverview[]>([]);
    const [appealState, setAppealState] = useState<AppealState>(
        AppealState.Any
    );
    const [loading, setLoading] = useState(false);
    const [totalRows] = useState<number>(0);

    useEffect(() => {
        const abortController = new AbortController();
        setLoading(true);
        apiGetAppeals(undefined, abortController)
            .then((response) => {
                setAppeals(response);
            })
            .catch(logErr)
            .finally(() => setLoading(false));

        return () => abortController.abort();
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

    const tableIcon = useMemo(() => {
        if (loading) {
            return <LoadingSpinner />;
        }
        switch (appealState) {
            case AppealState.Accepted:
                return <GppGoodIcon />;
            case AppealState.Open:
                return <FiberNewIcon />;
            case AppealState.Denied:
                return <DoNotDisturbIcon />;
            default:
                return <SnoozeIcon />;
        }
    }, [appealState, loading]);

    return (
        <Grid container spacing={3}>
            <Grid container spacing={2}>
                <Grid xs={6} md={3}>
                    <BasicStatCard
                        value={newAppeals}
                        title={'New/Open Appeals'}
                        desc={'Recently created & open appeals'}
                        icon={<FiberNewIcon />}
                    />
                </Grid>

                <Grid xs={6} md={3}>
                    <BasicStatCard
                        value={oldAppeals}
                        title={'Old/Open'}
                        desc={
                            'Appeals with no activity for >2days, but not resolved'
                        }
                        icon={<SnoozeIcon />}
                    />
                </Grid>

                <Grid xs={6} md={3}>
                    <BasicStatCard
                        value={deniedAppeals}
                        title={'Denied'}
                        desc={'Number of Denied & No Appeal '}
                        icon={<DoNotDisturbIcon />}
                    />
                </Grid>

                <Grid xs={6} md={3}>
                    <BasicStatCard
                        value={resolvedAppeals}
                        title={'Accepted'}
                        desc={'Users with accept appeals'}
                        icon={<GppGoodIcon />}
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
                    <LazyTablePaginator
                        page={page}
                        total={totalRows}
                        loading={loading}
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
                <ContainerWithHeader
                    title={'Recent Open Appeal Activity'}
                    iconLeft={tableIcon}
                >
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
                                        fullWidth
                                        component={Link}
                                        variant={'text'}
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
                                        avatar_hash={row.source_avatar}
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
                                        avatar_hash={row.target_avatar}
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
                                label: 'Last Activity',
                                tooltip:
                                    'Updated when a user sends/edits an appeal message',
                                sortable: true,
                                align: 'left',
                                width: '150px',
                                renderer: (obj) => {
                                    return (
                                        <Typography variant={'body1'}>
                                            {format(
                                                obj.updated_on,
                                                'yyyy-MM-dd HH:mm'
                                            )}
                                        </Typography>
                                    );
                                }
                            }
                        ]}
                    />
                </ContainerWithHeader>
            </Grid>
        </Grid>
    );
};
