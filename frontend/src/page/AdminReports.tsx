import React, { useCallback, useEffect, useState } from 'react';
import FilterListIcon from '@mui/icons-material/FilterList';
import ReportIcon from '@mui/icons-material/Report';
import Typography from '@mui/material/Typography';
import Grid from '@mui/material/Unstable_Grid2';
import { parseISO } from 'date-fns';
import format from 'date-fns/format';
import { Formik } from 'formik';
import * as yup from 'yup';
import {
    apiGetReports,
    BanReasons,
    ReportQueryFilter,
    ReportStatus,
    reportStatusString,
    ReportWithAuthor
} from '../api';
import { ContainerWithHeader } from '../component/ContainerWithHeader';
import { LazyTable, Order, RowsPerPage } from '../component/LazyTable';
import { LoadingIcon } from '../component/LoadingIcon';
import { TableCellLink } from '../component/TableCellLink';
import { FilterButtons } from '../component/formik/FilterButtons';
import {
    ReportStatusField,
    reportStatusFielValidator
} from '../component/formik/ReportStatusField';
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

interface FilterValues {
    report_status: ReportStatus;
    source_id: string;
    target_id: string;
}

const validationSchema = yup.object({
    report_status: reportStatusFielValidator,
    source_id: sourceIdValidator,
    target_id: targetIdValidator
});

export const AdminReports = () => {
    const [reports, setReports] = useState<ReportWithAuthor[]>([]);
    const [filterStatus, setFilterStatus] = useState(ReportStatus.Any);
    const [sortOrder, setSortOrder] = useState<Order>('desc');
    const [sortColumn, setSortColumn] =
        useState<keyof ReportWithAuthor>('report_id');
    const [rowPerPageCount, setRowPerPageCount] = useState<number>(
        RowsPerPage.Fifty
    );
    const [page, setPage] = useState(0);
    const [loading, setLoading] = useState(false);
    const [totalRows, setTotalRows] = useState<number>(0);
    const [author, setAuthor] = useState('');
    const [target, setTarget] = useState('');

    useEffect(() => {
        const opts: ReportQueryFilter = {
            limit: rowPerPageCount,
            offset: page * rowPerPageCount,
            order_by: sortColumn,
            desc: sortOrder == 'desc',
            report_status: filterStatus,
            source_id: author,
            target_id: target
        };
        setLoading(true);
        apiGetReports(opts)
            .then((resp) => {
                setReports(resp.data || []);
                setTotalRows(resp.count);
                if (page * rowPerPageCount > resp.count) {
                    setPage(0);
                }
            })
            .catch((e) => {
                logErr(e);
            })
            .finally(() => {
                setLoading(false);
            });
    }, [
        author,
        filterStatus,
        page,
        rowPerPageCount,
        sortColumn,
        sortOrder,
        target
    ]);

    const onFilterSubmit = useCallback((values: FilterValues) => {
        setAuthor(values.source_id);
        setTarget(values.target_id);
        setFilterStatus(values.report_status);
    }, []);

    const onFilterReset = useCallback(() => {
        setAuthor('');
        setTarget('');
        setFilterStatus(ReportStatus.Any);
    }, []);

    return (
        <Formik<FilterValues>
            onSubmit={onFilterSubmit}
            onReset={onFilterReset}
            initialValues={{
                report_status: filterStatus,
                source_id: author,
                target_id: target
            }}
            validationSchema={validationSchema}
            validateOnChange={true}
            validateOnBlur={true}
        >
            <Grid container spacing={2}>
                <Grid xs={12}>
                    <ContainerWithHeader
                        title={'Filters'}
                        iconLeft={<FilterListIcon />}
                    >
                        <Grid container spacing={2}>
                            <Grid xs>
                                <SourceIdField />
                            </Grid>
                            <Grid xs>
                                <TargetIDField />
                            </Grid>
                            <Grid xs>
                                <ReportStatusField />
                            </Grid>
                            <Grid xs>
                                <FilterButtons />
                            </Grid>
                        </Grid>
                    </ContainerWithHeader>
                </Grid>
                <Grid xs={12}>
                    <ContainerWithHeader
                        title={'Current User Reports'}
                        iconLeft={loading ? <LoadingIcon /> : <ReportIcon />}
                    >
                        <LazyTable
                            showPager={true}
                            count={totalRows}
                            rows={reports}
                            page={page}
                            rowsPerPage={rowPerPageCount}
                            sortOrder={sortOrder}
                            sortColumn={sortColumn}
                            onSortColumnChanged={async (column) => {
                                setSortColumn(column);
                            }}
                            onSortOrderChanged={async (direction) => {
                                setSortOrder(direction);
                            }}
                            onPageChange={(_, newPage: number) => {
                                setPage(newPage);
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
                            columns={[
                                {
                                    label: 'ID',
                                    tooltip: 'Report ID',
                                    sortType: 'number',
                                    sortKey: 'report_id',
                                    sortable: true,
                                    align: 'left',
                                    renderer: (obj) => (
                                        <TableCellLink
                                            to={`/report/${obj.report_id}`}
                                            label={`#${obj.report_id}`}
                                        />
                                    )
                                },
                                {
                                    label: 'Status',
                                    tooltip: 'Status',
                                    sortKey: 'report_status',
                                    sortable: true,
                                    align: 'left',
                                    width: '200px',
                                    renderer: (obj) => {
                                        return (
                                            <Typography variant={'subtitle1'}>
                                                {reportStatusString(
                                                    obj.report_status
                                                )}
                                            </Typography>
                                        );
                                    }
                                },
                                {
                                    label: 'Reporter',
                                    tooltip: 'Reporter',
                                    sortType: 'string',
                                    align: 'left',
                                    renderer: (row) => (
                                        <SteamIDSelectField
                                            steam_id={row.author.steam_id}
                                            personaname={
                                                row.author.personaname ||
                                                row.source_id
                                            }
                                            avatarhash={row.author.avatarhash}
                                            field_name={'source_id'}
                                        />
                                    )
                                },
                                {
                                    label: 'Subject',
                                    tooltip: 'Subject',
                                    sortType: 'string',
                                    align: 'left',
                                    width: '250px',
                                    renderer: (row) => (
                                        <SteamIDSelectField
                                            steam_id={row.subject.steam_id}
                                            personaname={
                                                row.subject.personaname ||
                                                row.target_id
                                            }
                                            avatarhash={row.subject.avatarhash}
                                            field_name={'target_id'}
                                        />
                                    )
                                },
                                {
                                    label: 'Reason',
                                    tooltip: 'Reason For Report',
                                    sortType: 'number',
                                    sortKey: 'reason',
                                    align: 'left',
                                    width: '250px',
                                    renderer: (row) => (
                                        <Typography variant={'body1'}>
                                            {BanReasons[row.reason]}
                                        </Typography>
                                    )
                                },
                                {
                                    label: 'Created',
                                    tooltip: 'Created On',
                                    sortType: 'date',
                                    align: 'left',
                                    width: '150px',
                                    sortable: true,
                                    sortKey: 'created_on',
                                    renderer: (obj) => {
                                        return (
                                            <Typography variant={'body1'}>
                                                {format(
                                                    parseISO(
                                                        obj.created_on as never as string
                                                    ),
                                                    'yyyy-MM-dd HH:mm'
                                                )}
                                            </Typography>
                                        );
                                    }
                                },
                                {
                                    label: 'Last Activity',
                                    tooltip: 'Last Activity',
                                    sortType: 'date',
                                    align: 'left',
                                    width: '150px',
                                    sortable: true,
                                    sortKey: 'updated_on',
                                    renderer: (obj) => {
                                        return (
                                            <Typography variant={'body1'}>
                                                {format(
                                                    parseISO(
                                                        obj.updated_on as never as string
                                                    ),
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
        </Formik>
    );
};
