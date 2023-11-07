import React, { useCallback, useEffect, useState } from 'react';
import CheckIcon from '@mui/icons-material/Check';
import FilterListIcon from '@mui/icons-material/FilterList';
import ReportIcon from '@mui/icons-material/Report';
import Stack from '@mui/material/Stack';
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
import { Order, RowsPerPage } from '../component/DataTable';
import { LazyTable } from '../component/LazyTable';
import { LoadingSpinner } from '../component/LoadingSpinner';
import { PersonCell } from '../component/PersonCell';
import { TableCellLink } from '../component/TableCellLink';
import {
    AuthorIDField,
    authorIdValidator
} from '../component/formik/AuthorIdField';
import {
    ReportStatusField,
    reportStatusFielValidator
} from '../component/formik/ReportStatusField';
import {
    TargetIDField,
    targetIdValidator
} from '../component/formik/TargetIdField';
import { ResetButton, SubmitButton } from '../component/modal/Buttons';
import { logErr } from '../util/errors';

interface FilterValues {
    report_status: ReportStatus;
    author_id: string;
    target_id: string;
}

const validationSchema = yup.object({
    report_status: reportStatusFielValidator,
    author_id: authorIdValidator,
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
        const opts: ReportQueryFilter<ReportWithAuthor> = {
            limit: rowPerPageCount,
            offset: page * rowPerPageCount,
            order_by: sortColumn,
            desc: sortOrder == 'desc',
            report_status: filterStatus,
            author_id: author,
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

    const onFilterSumbit = useCallback((values: FilterValues) => {
        setAuthor(values.author_id);
        setTarget(values.target_id);
        setFilterStatus(values.report_status);
    }, []);

    const onFilterReset = useCallback(() => {
        setAuthor('');
        setTarget('');
        setFilterStatus(ReportStatus.Any);
    }, []);

    return (
        <Grid container spacing={2}>
            <Grid xs={12}>
                <ContainerWithHeader
                    title={'Filters'}
                    iconLeft={<FilterListIcon />}
                >
                    <Formik<FilterValues>
                        onSubmit={onFilterSumbit}
                        onReset={onFilterReset}
                        initialValues={{
                            report_status: filterStatus,
                            author_id: author,
                            target_id: target
                        }}
                        validationSchema={validationSchema}
                        validateOnChange={true}
                        validateOnBlur={true}
                    >
                        <Grid container>
                            <Grid xs={12} padding={2}>
                                <Stack direction={'row'} spacing={2}>
                                    <ReportStatusField />
                                    <AuthorIDField />
                                    <TargetIDField />
                                </Stack>
                            </Grid>
                            <Grid xs={12} padding={2}>
                                <Stack
                                    direction={'row'}
                                    spacing={2}
                                    flexDirection={'row-reverse'}
                                >
                                    <SubmitButton
                                        label={'Apply'}
                                        startIcon={<CheckIcon />}
                                    />
                                    <ResetButton />
                                </Stack>
                            </Grid>
                        </Grid>
                    </Formik>
                </ContainerWithHeader>
            </Grid>
            <Grid xs={12}>
                <ContainerWithHeader
                    title={'Current User Reports'}
                    iconLeft={loading ? <LoadingSpinner /> : <ReportIcon />}
                >
                    <LazyTable
                        showPager={true}
                        count={totalRows}
                        rows={reports}
                        loading={loading}
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
                                queryValue: (o) => `${o.report_id}`,
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
                                queryValue: (o) =>
                                    reportStatusString(o.report_status),
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
                                queryValue: (o) =>
                                    o.subject.personaname + o.subject.steam_id,
                                renderer: (row) => (
                                    <PersonCell
                                        steam_id={row.author.steam_id}
                                        personaname={row.author.personaname}
                                        avatar_hash={row.author.avatar}
                                    ></PersonCell>
                                )
                            },
                            {
                                label: 'Subject',
                                tooltip: 'Subject',
                                sortType: 'string',
                                align: 'left',
                                width: '250px',
                                queryValue: (o) =>
                                    o.subject.personaname + o.subject.steam_id,
                                renderer: (row) => (
                                    <PersonCell
                                        steam_id={row.subject.steam_id}
                                        personaname={row.subject.personaname}
                                        avatar_hash={row.subject.avatar}
                                    ></PersonCell>
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
    );
};
