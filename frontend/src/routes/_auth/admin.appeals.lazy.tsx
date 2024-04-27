import { ChangeEvent, useCallback, useMemo } from 'react';
import useUrlState from '@ahooksjs/use-url-state';
import DoNotDisturbIcon from '@mui/icons-material/DoNotDisturb';
import FiberNewIcon from '@mui/icons-material/FiberNew';
import FilterListIcon from '@mui/icons-material/FilterList';
import GppGoodIcon from '@mui/icons-material/GppGood';
import SnoozeIcon from '@mui/icons-material/Snooze';
import Typography from '@mui/material/Typography';
import Grid from '@mui/material/Unstable_Grid2';
import { createLazyFileRoute } from '@tanstack/react-router';
import { Formik } from 'formik';
import * as yup from 'yup';
import { AppealState, appealStateString, BanReason, SteamBanRecord } from '../../api';
import { ContainerWithHeader } from '../../component/ContainerWithHeader.tsx';
import { LoadingIcon } from '../../component/LoadingIcon.tsx';
import { LoadingSpinner } from '../../component/LoadingSpinner.tsx';
import { AppealStateField } from '../../component/formik/AppealStateField.tsx';
import { FilterButtons } from '../../component/formik/FilterButtons.tsx';
import { SourceIDField } from '../../component/formik/SourceIDField.tsx';
import { SteamIDSelectField } from '../../component/formik/SteamIDSelectField.tsx';
import { TargetIDField } from '../../component/formik/TargetIdField.tsx';
import { LazyTable } from '../../component/table/LazyTable.tsx';
import { TableCellLink } from '../../component/table/TableCellLink.tsx';
import { useAppeals } from '../../hooks/useAppeals.ts';
import { RowsPerPage } from '../../util/table.ts';
import { renderDate, renderDateTime } from '../../util/text.tsx';
import { appealStateFielValidator, sourceIdValidator, targetIdValidator } from '../../util/validators.ts';

export const Route = createLazyFileRoute('/_auth/admin/appeals')({
    component: AdminAppeals
});

interface AppealFilterValues {
    appeal_state?: AppealState;
    source_id?: string;
    target_id?: string;
}

const validationSchema = yup.object({
    appeal_state: appealStateFielValidator,
    source_id: sourceIdValidator,
    target_id: targetIdValidator
});

function AdminAppeals() {
    const [state, setState] = useUrlState({
        page: undefined,
        source: undefined,
        target: undefined,
        appealState: undefined,
        rows: undefined,
        sortOrder: undefined,
        sortColumn: undefined
    });

    const { appeals, count, loading } = useAppeals({
        limit: Number(state.rows ?? RowsPerPage.Ten),
        offset: Number((state.page ?? 0) * (state.rows ?? RowsPerPage.Ten)),
        order_by: state.sortColumn ?? 'ban_id',
        desc: (state.sortOrder ?? 'desc') == 'desc',
        source_id: state.source ?? '',
        target_id: state.target ?? '',
        appeal_state: Number(state.appealState ?? AppealState.Any)
    });

    const tableIcon = useMemo(() => {
        if (loading) {
            return <LoadingSpinner />;
        }
        switch (state.appealState) {
            case AppealState.Accepted:
                return <GppGoodIcon />;
            case AppealState.Open:
                return <FiberNewIcon />;
            case AppealState.Denied:
                return <DoNotDisturbIcon />;
            default:
                return <SnoozeIcon />;
        }
    }, [loading, state.appealState]);

    const onSubmit = useCallback(
        (values: AppealFilterValues) => {
            setState({
                appealState: values.appeal_state != AppealState.Any ? values.appeal_state : undefined,
                source: values.source_id != '' ? values.source_id : undefined,
                target: values.target_id != '' ? values.target_id : undefined
            });
        },
        [setState]
    );

    const onReset = useCallback(() => {
        setState({
            appealState: undefined,
            source: undefined,
            target: undefined
        });
    }, [setState]);

    return (
        <Formik<AppealFilterValues>
            initialValues={{
                appeal_state: Number(state.appealState ?? AppealState.Any),
                source_id: state.source,
                target_id: state.target
            }}
            onReset={onReset}
            onSubmit={onSubmit}
            validationSchema={validationSchema}
            validateOnChange={true}
        >
            <Grid container spacing={3}>
                <Grid xs={12}>
                    <ContainerWithHeader title={'Appeal Activity Filters'} iconLeft={<FilterListIcon />}>
                        <Grid container spacing={2}>
                            <Grid xs={6} sm={4} md={3}>
                                <AppealStateField />
                            </Grid>
                            <Grid xs={6} sm={4} md={3}>
                                <SourceIDField />
                            </Grid>
                            <Grid xs={6} sm={4} md={3}>
                                <TargetIDField />
                            </Grid>
                            <Grid xs={6} sm={4} md={3}>
                                <FilterButtons />
                            </Grid>
                        </Grid>
                    </ContainerWithHeader>
                </Grid>

                <Grid xs={12}>
                    <ContainerWithHeader title={'Recent Open Appeal Activity'} iconLeft={loading ? <LoadingIcon /> : tableIcon}>
                        <LazyTable<SteamBanRecord>
                            rows={appeals}
                            showPager
                            page={Number(state.page ?? 0)}
                            rowsPerPage={Number(state.rows ?? RowsPerPage.Ten)}
                            count={count}
                            sortOrder={state.sortOrder}
                            sortColumn={state.sortColumn}
                            onSortColumnChanged={async (column) => {
                                setState({ sortColumn: column });
                            }}
                            onSortOrderChanged={async (direction) => {
                                setState({ sortOrder: direction });
                            }}
                            onRowsPerPageChange={(event: ChangeEvent<HTMLInputElement | HTMLTextAreaElement>) => {
                                setState({
                                    rows: Number(event.target.value),
                                    page: 0
                                });
                            }}
                            onPageChange={(_, newPage) => {
                                setState({ page: newPage });
                            }}
                            columns={[
                                {
                                    label: '#',
                                    tooltip: 'Ban ID',
                                    sortable: true,
                                    align: 'left',
                                    renderer: (obj) => <TableCellLink label={`#${obj.ban_id}`} to={`/ban/${obj.ban_id}`} />
                                },
                                {
                                    label: 'Appeal',
                                    tooltip: 'Appeal State',
                                    sortable: true,
                                    sortKey: 'appeal_state',
                                    align: 'left',
                                    renderer: (row) => <Typography variant={'body1'}>{appealStateString(row.appeal_state)}</Typography>
                                },
                                {
                                    label: 'Author',
                                    tooltip: 'Author',
                                    sortable: true,
                                    align: 'left',
                                    renderer: (row) => (
                                        <SteamIDSelectField
                                            steam_id={row.source_id}
                                            personaname={row.source_personaname || row.source_id}
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
                                            personaname={row.target_personaname || row.target_id}
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
                                    renderer: (row) => <Typography variant={'body1'}>{BanReason[row.reason]}</Typography>
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
                                        return <Typography variant={'body1'}>{renderDate(obj.created_on)}</Typography>;
                                    }
                                },
                                {
                                    label: 'Last Activity',
                                    tooltip: 'Updated when a user sends/edits an appeal message',
                                    sortable: true,
                                    sortKey: 'updated_on',
                                    align: 'left',
                                    width: '150px',
                                    renderer: (obj) => {
                                        return <Typography variant={'body1'}>{renderDateTime(obj.updated_on)}</Typography>;
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
