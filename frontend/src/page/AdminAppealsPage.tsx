import React, { useCallback, useEffect, useMemo, useState } from 'react';
import DoNotDisturbIcon from '@mui/icons-material/DoNotDisturb';
import FiberNewIcon from '@mui/icons-material/FiberNew';
import FilterListIcon from '@mui/icons-material/FilterList';
import GppGoodIcon from '@mui/icons-material/GppGood';
import SnoozeIcon from '@mui/icons-material/Snooze';
import Typography from '@mui/material/Typography';
import Grid from '@mui/material/Unstable_Grid2';
import { Formik } from 'formik';
import * as yup from 'yup';
import {
    apiGetAppeals,
    AppealQueryFilter,
    AppealState,
    appealStateString,
    BanReason,
    SteamBanRecord
} from '../api';
import { ContainerWithHeader } from '../component/ContainerWithHeader';
import { LazyTable, Order, RowsPerPage } from '../component/LazyTable';
import { LoadingIcon } from '../component/LoadingIcon';
import { LoadingSpinner } from '../component/LoadingSpinner';
import { TableCellLink } from '../component/TableCellLink';
import {
    AppealStateField,
    appealStateFielValidator
} from '../component/formik/AppealStateField';
import { FilterButtons } from '../component/formik/FilterButtons';
import {
    SourceIdField,
    sourceIdValidator
} from '../component/formik/SourceIdField';
import { SteamIDSelectField } from '../component/formik/SteamIDSelectField';
import {
    TargetIDField,
    targetIdValidator
} from '../component/formik/TargetIdField';
import { logErr } from '../util/errors';
import { renderDate, renderDateTime } from '../util/text';

interface AppealFilterValues {
    appeal_state: AppealState;
    source_id: string;
    target_id: string;
}

const validationSchema = yup.object({
    appeal_state: appealStateFielValidator,
    source_id: sourceIdValidator,
    target_id: targetIdValidator
});

export const AdminAppealsPage = () => {
    const [sortOrder, setSortOrder] = useState<Order>('desc');
    const [sortColumn, setSortColumn] =
        useState<keyof SteamBanRecord>('ban_id');
    const [rowPerPageCount, setRowPerPageCount] = useState<number>(
        RowsPerPage.Fifty
    );
    const [page, setPage] = useState(0);
    const [appeals, setAppeals] = useState<SteamBanRecord[]>([]);
    const [appealState, setAppealState] = useState<AppealState>(
        AppealState.Any
    );
    const [loading, setLoading] = useState(false);
    const [totalRows, setTotalRows] = useState<number>(0);
    const [author, setAuthor] = useState('');
    const [target, setTarget] = useState('');

    useEffect(() => {
        const abortController = new AbortController();

        const opts: AppealQueryFilter = {
            desc: sortOrder == 'desc',
            order_by: sortColumn,
            source_id: author,
            target_id: target,
            offset: page,
            limit: rowPerPageCount,
            appeal_state: appealState
        };

        setLoading(true);
        apiGetAppeals(opts, abortController)
            .then((response) => {
                setAppeals(response.data);
                setTotalRows(response.count);
            })
            .catch(logErr)
            .finally(() => setLoading(false));

        return () => abortController.abort();
    }, [
        appealState,
        author,
        page,
        rowPerPageCount,
        sortColumn,
        sortOrder,
        target
    ]);

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

    const onSubmit = useCallback((values: AppealFilterValues) => {
        setAppealState(values.appeal_state);
        setAuthor(values.source_id);
        setTarget(values.target_id);
    }, []);

    const onReset = useCallback(() => {
        setAppealState(AppealState.Any);
        setAuthor('');
        setTarget('');
    }, []);

    return (
        <Formik<AppealFilterValues>
            initialValues={{
                appeal_state: appealState,
                source_id: author,
                target_id: target
            }}
            onReset={onReset}
            onSubmit={onSubmit}
            validationSchema={validationSchema}
            validateOnChange={true}
        >
            <Grid container spacing={3}>
                <Grid xs={12}>
                    <ContainerWithHeader
                        title={'Appeal Activity Filters'}
                        iconLeft={<FilterListIcon />}
                    >
                        <Grid container spacing={2}>
                            <Grid xs>
                                <AppealStateField />
                            </Grid>
                            <Grid xs>
                                <SourceIdField />
                            </Grid>
                            <Grid xs>
                                <TargetIDField />
                            </Grid>
                            <Grid xs>
                                <FilterButtons />
                            </Grid>
                        </Grid>
                    </ContainerWithHeader>
                </Grid>

                <Grid xs={12}>
                    <ContainerWithHeader
                        title={'Recent Open Appeal Activity'}
                        iconLeft={loading ? <LoadingIcon /> : tableIcon}
                    >
                        <LazyTable<SteamBanRecord>
                            rows={appeals}
                            showPager
                            page={page}
                            rowsPerPage={rowPerPageCount}
                            count={totalRows}
                            sortOrder={sortOrder}
                            sortColumn={sortColumn}
                            onSortColumnChanged={async (column) => {
                                setSortColumn(column);
                            }}
                            onSortOrderChanged={async (direction) => {
                                setSortOrder(direction);
                            }}
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
                            columns={[
                                {
                                    label: '#',
                                    tooltip: 'Ban ID',
                                    sortable: true,
                                    align: 'left',
                                    renderer: (obj) => (
                                        <TableCellLink
                                            label={`#${obj.ban_id}`}
                                            to={`/ban/${obj.ban_id}`}
                                        />
                                    )
                                },
                                {
                                    label: 'Appeal',
                                    tooltip: 'Appeal State',
                                    sortable: true,
                                    sortKey: 'appeal_state',
                                    align: 'left',
                                    renderer: (row) => (
                                        <Typography variant={'body1'}>
                                            {appealStateString(
                                                row.appeal_state
                                            )}
                                        </Typography>
                                    )
                                },
                                {
                                    label: 'Author',
                                    tooltip: 'Author',
                                    sortable: true,
                                    align: 'left',
                                    renderer: (row) => (
                                        <SteamIDSelectField
                                            steam_id={row.source_id}
                                            personaname={
                                                row.source_personaname ||
                                                row.source_id
                                            }
                                            avatarhash={row.source_avatarhash}
                                            field_name={'source_id'}
                                        />
                                    )
                                },
                                {
                                    label: 'Target',
                                    tooltip: 'Target',
                                    sortable: true,
                                    align: 'left',
                                    renderer: (row) => (
                                        <SteamIDSelectField
                                            steam_id={row.target_id}
                                            personaname={
                                                row.target_personaname ||
                                                row.target_id
                                            }
                                            avatarhash={row.target_avatarhash}
                                            field_name={'target_id'}
                                        />
                                    )
                                },
                                {
                                    label: 'Reason',
                                    tooltip: 'Reason',
                                    sortKey: 'reason',
                                    sortable: true,
                                    align: 'left',
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
                                                {renderDate(obj.created_on)}
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
                                                {renderDateTime(obj.updated_on)}
                                            </Typography>
                                        );
                                    }
                                }
                            ]}
                        />
                    </ContainerWithHeader>
                </Grid>
            </Grid>
        </Formik>
    );
};