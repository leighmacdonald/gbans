import { ChangeEvent, useCallback } from 'react';
import useUrlState from '@ahooksjs/use-url-state';
import FilterListIcon from '@mui/icons-material/FilterList';
import ReportIcon from '@mui/icons-material/Report';
import Typography from '@mui/material/Typography';
import Grid from '@mui/material/Unstable_Grid2';
import { createLazyFileRoute } from '@tanstack/react-router';
import { parseISO } from 'date-fns';
import { Formik } from 'formik';
import * as yup from 'yup';
import { BanReasons, ReportStatus, reportStatusString } from '../api';
import { ContainerWithHeader } from '../component/ContainerWithHeader';
import { LoadingIcon } from '../component/LoadingIcon';
import { FilterButtons } from '../component/formik/FilterButtons';
import { ReportStatusField } from '../component/formik/ReportStatusField';
import { SourceIDField } from '../component/formik/SourceIDField.tsx';
import { SteamIDSelectField } from '../component/formik/SteamIDSelectField';
import { TargetIDField } from '../component/formik/TargetIdField';
import { LazyTable } from '../component/table/LazyTable';
import { TableCellLink } from '../component/table/TableCellLink';
import { useReports } from '../hooks/useReports';
import { RowsPerPage } from '../util/table.ts';
import { renderDateTime } from '../util/text.tsx';
import {
    reportStatusFieldValidator,
    sourceIdValidator,
    targetIdValidator
} from '../util/validators.ts';

export const Route = createLazyFileRoute('/admin/reports')({
    component: AdminReports
});

interface FilterValues {
    report_status: ReportStatus;
    source_id: string;
    target_id: string;
}

const validationSchema = yup.object({
    report_status: reportStatusFieldValidator,
    source_id: sourceIdValidator,
    target_id: targetIdValidator
});

function AdminReports() {
    const [state, setState] = useUrlState({
        page: undefined,
        source: undefined,
        target: undefined,
        reportStatus: undefined,
        rows: undefined,
        sortOrder: undefined,
        sortColumn: undefined
    });

    const { data, count, loading } = useReports({
        limit: Number(state.rows ?? RowsPerPage.Ten),
        offset: Number((state.page ?? 0) * (state.rows ?? RowsPerPage.Ten)),
        order_by: state.sortColumn ?? 'report_id',
        desc: (state.sortOrder ?? 'desc') == 'desc',
        source_id: state.source ?? '',
        target_id: state.target ?? '',
        report_status: Number(state.reportStatus ?? ReportStatus.Any)
    });

    const onFilterSubmit = useCallback(
        (values: FilterValues) => {
            setState({
                reportStatus:
                    values.report_status != ReportStatus.Any
                        ? values.report_status
                        : undefined,
                source: values.source_id != '' ? values.source_id : undefined,
                target: values.target_id != '' ? values.target_id : undefined
            });
        },
        [setState]
    );

    const onFilterReset = useCallback(() => {
        setState({
            reportStatus: undefined,
            source: undefined,
            target: undefined
        });
    }, [setState]);

    return (
        <Formik<FilterValues>
            onSubmit={onFilterSubmit}
            onReset={onFilterReset}
            initialValues={{
                report_status: Number(state.reportStatus ?? ReportStatus.Any),
                source_id: state.source,
                target_id: state.target
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
                            <Grid xs={4} sm={4} md={3}>
                                <SourceIDField />
                            </Grid>
                            <Grid xs={4} sm={4} md={3}>
                                <TargetIDField />
                            </Grid>
                            <Grid xs={4} sm={4} md={3}>
                                <ReportStatusField />
                            </Grid>
                            <Grid xs={4} sm={4} md={3}>
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
                            count={count}
                            rows={data}
                            page={Number(state.page ?? 0)}
                            rowsPerPage={Number(state.rows ?? RowsPerPage.Ten)}
                            sortOrder={state.sortOrder}
                            sortColumn={state.sortColumn}
                            onSortColumnChanged={async (column) => {
                                setState({ sortColumn: column });
                            }}
                            onSortOrderChanged={async (direction) => {
                                setState({ sortOrder: direction });
                            }}
                            onPageChange={(_, newPage: number) => {
                                setState({ page: newPage });
                            }}
                            onRowsPerPageChange={(
                                event: ChangeEvent<
                                    HTMLInputElement | HTMLTextAreaElement
                                >
                            ) => {
                                setState({
                                    rows: Number(event.target.value),
                                    page: 0
                                });
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
                                                {renderDateTime(
                                                    parseISO(
                                                        obj.created_on as never as string
                                                    )
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
                                                {renderDateTime(
                                                    parseISO(
                                                        obj.updated_on as never as string
                                                    )
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
}
