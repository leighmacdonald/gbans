import FilterListIcon from '@mui/icons-material/FilterList';
import VisibilityIcon from '@mui/icons-material/Visibility';
import Link from '@mui/material/Link';
import MenuItem from '@mui/material/MenuItem';
import TableCell from '@mui/material/TableCell';
import Tooltip from '@mui/material/Tooltip';
import Typography from '@mui/material/Typography';
import Grid from '@mui/material/Unstable_Grid2';
import { useForm } from '@tanstack/react-form';
import { useQuery } from '@tanstack/react-query';
import { createFileRoute, useNavigate } from '@tanstack/react-router';
import { createColumnHelper, getCoreRowModel, useReactTable } from '@tanstack/react-table';
import { zodValidator } from '@tanstack/zod-form-adapter';
import { z } from 'zod';
import { apiGetAppeals, AppealState, AppealStateCollection, appealStateString, BanReasons, SteamBanRecord } from '../api';
import { ContainerWithHeader } from '../component/ContainerWithHeader.tsx';
import { DataTable } from '../component/DataTable.tsx';
import { Paginator } from '../component/Paginator.tsx';
import { PersonCell } from '../component/PersonCell.tsx';
import RouterLink from '../component/RouterLink.tsx';
import { TableHeadingCell } from '../component/TableHeadingCell.tsx';
import { Buttons } from '../component/field/Buttons.tsx';
import { SelectFieldSimple } from '../component/field/SelectFieldSimple.tsx';
import { makeSteamidValidatorsOptional } from '../component/field/SteamIDField.tsx';
import { TextFieldSimple } from '../component/field/TextFieldSimple.tsx';
import { commonTableSearchSchema, LazyResult, RowsPerPage } from '../util/table.ts';
import { renderDateTime } from '../util/text.tsx';

const appealSearchSchema = z.object({
    ...commonTableSearchSchema,
    sortColumn: z.enum(['report_id', 'source_id', 'target_id', 'appeal_state', 'reason', 'created_on', 'updated_on']).optional(),
    source_id: z.string().optional(),
    target_id: z.string().optional(),
    appeal_state: z.nativeEnum(AppealState).optional()
});

export const Route = createFileRoute('/_mod/admin/appeals')({
    component: AdminAppeals,
    validateSearch: (search) => appealSearchSchema.parse(search)
});

function AdminAppeals() {
    const defaultRows = RowsPerPage.TwentyFive;
    const navigate = useNavigate({ from: Route.fullPath });
    const { page, sortColumn, rows, sortOrder, source_id, target_id, appeal_state } = Route.useSearch();
    const { data: appeals, isLoading } = useQuery({
        queryKey: ['appeals', { page, rows, sortOrder, appeal_state, source_id, target_id }],
        queryFn: async () => {
            return await apiGetAppeals({
                limit: rows ?? defaultRows,
                offset: (page ?? 0) * (rows ?? defaultRows),
                order_by: sortColumn ?? 'ban_id',
                desc: (sortOrder ?? 'desc') == 'desc',
                source_id: source_id ?? '',
                target_id: target_id ?? '',
                appeal_state: Number(appeal_state ?? AppealState.Any)
            });
        }
    });
    const { Field, Subscribe, handleSubmit, reset } = useForm({
        onSubmit: async ({ value }) => {
            await navigate({ to: '/admin/appeals', search: (prev) => ({ ...prev, ...value }) });
        },
        validatorAdapter: zodValidator,
        validators: {
            onChange: appealSearchSchema
        },
        defaultValues: {
            source_id: source_id ?? '',
            target_id: target_id ?? '',
            appeal_state: appeal_state ?? AppealState.Any
        }
    });

    const clear = async () => {
        await navigate({
            to: '/admin/appeals',
            search: (prev) => ({ ...prev, source_id: undefined, target_id: undefined, appeal_state: undefined })
        });
    };

    // const tableIcon = useMemo(() => {
    //     if (loading) {
    //         return <LoadingSpinner />;
    //     }
    //     switch (state.appealState) {
    //         case AppealState.Accepted:
    //             return <GppGoodIcon />;
    //         case AppealState.Open:
    //             return <FiberNewIcon />;
    //         case AppealState.Denied:
    //             return <DoNotDisturbIcon />;
    //         default:
    //             return <SnoozeIcon />;
    //     }
    // }, [loading, state.appealState]);
    //
    // const onSubmit = useCallback(
    //     (values: AppealFilterValues) => {
    //         setState({
    //             appealState: values.appeal_state != AppealState.Any ? values.appeal_state : undefined,
    //             source: values.source_id != '' ? values.source_id : undefined,
    //             target: values.target_id != '' ? values.target_id : undefined
    //         });
    //     },
    //     [setState]
    // );
    //
    // const onReset = useCallback(() => {
    //     setState({
    //         appealState: undefined,
    //         source: undefined,
    //         target: undefined
    //     });
    // }, [setState]);

    return (
        <Grid container spacing={2}>
            <Grid xs={12}>
                <ContainerWithHeader title={'Filters'} iconLeft={<FilterListIcon />} marginTop={2}>
                    <form
                        onSubmit={async (e) => {
                            e.preventDefault();
                            e.stopPropagation();
                            await handleSubmit();
                        }}
                    >
                        <Grid container spacing={2}>
                            <Grid xs={6} md={4}>
                                <Field
                                    name={'source_id'}
                                    validators={makeSteamidValidatorsOptional()}
                                    children={(props) => {
                                        return <TextFieldSimple {...props} label={'Author Steam ID'} fullwidth={true} />;
                                    }}
                                />
                            </Grid>

                            <Grid xs={6} md={4}>
                                <Field
                                    name={'target_id'}
                                    validators={makeSteamidValidatorsOptional()}
                                    children={(props) => {
                                        return <TextFieldSimple {...props} label={'Subject Steam ID'} fullwidth={true} />;
                                    }}
                                />
                            </Grid>

                            <Grid xs={6} md={4}>
                                <Field
                                    name={'appeal_state'}
                                    children={(props) => {
                                        return (
                                            <SelectFieldSimple
                                                {...props}
                                                label={'Appeal Status'}
                                                fullwidth={true}
                                                items={AppealStateCollection}
                                                renderMenu={(item) => {
                                                    return (
                                                        <MenuItem value={item} key={`rs-${item}`}>
                                                            {appealStateString(item as AppealState)}
                                                        </MenuItem>
                                                    );
                                                }}
                                            />
                                        );
                                    }}
                                />
                            </Grid>
                            <Grid xs={12} mdOffset="auto">
                                <Subscribe
                                    selector={(state) => [state.canSubmit, state.isSubmitting]}
                                    children={([canSubmit, isSubmitting]) => (
                                        <Buttons reset={reset} canSubmit={canSubmit} isSubmitting={isSubmitting} onClear={clear} />
                                    )}
                                />
                            </Grid>
                        </Grid>
                    </form>
                </ContainerWithHeader>
            </Grid>

            <Grid xs={12}>
                <ContainerWithHeader title={'Recent Open Appeal Activity'}>
                    <AppealsTable appeals={appeals ?? { data: [], count: 0 }} isLoading={isLoading} />
                    <Paginator page={page ?? 0} rows={rows ?? defaultRows} path={'/admin/appeals'} />
                </ContainerWithHeader>
            </Grid>
        </Grid>
        // </Formik>
    );
}
const columnHelper = createColumnHelper<SteamBanRecord>();

const AppealsTable = ({ appeals, isLoading }: { appeals: LazyResult<SteamBanRecord>; isLoading: boolean }) => {
    const columns = [
        columnHelper.accessor('ban_id', {
            header: () => <TableHeadingCell name={'View'} />,
            cell: (info) => (
                <Link color={'primary'} component={RouterLink} to={`/ban/$ban_id`} params={{ ban_id: info.getValue() }}>
                    <Tooltip title={'View'}>
                        <VisibilityIcon />
                    </Tooltip>
                </Link>
            )
        }),
        columnHelper.accessor('appeal_state', {
            header: () => <TableHeadingCell name={'Status'} />,
            cell: (info) => {
                return (
                    <TableCell>
                        <Typography variant={'body1'}>{appealStateString(info.getValue())}</Typography>
                    </TableCell>
                );
            }
        }),
        columnHelper.accessor('source_id', {
            header: () => <TableHeadingCell name={'Author'} />,
            cell: (info) => (
                <PersonCell
                    steam_id={appeals.data[info.row.index].source_id}
                    personaname={appeals.data[info.row.index].source_personaname}
                    avatar_hash={appeals.data[info.row.index].source_avatarhash}
                />
            )
        }),
        columnHelper.accessor('target_id', {
            header: () => <TableHeadingCell name={'Subject'} />,
            cell: (info) => (
                <PersonCell
                    steam_id={appeals.data[info.row.index].target_id}
                    personaname={appeals.data[info.row.index].target_personaname}
                    avatar_hash={appeals.data[info.row.index].target_avatarhash}
                />
            )
        }),
        columnHelper.accessor('reason', {
            header: () => <TableHeadingCell name={'Reason'} />,
            cell: (info) => <Typography>{BanReasons[info.getValue()]}</Typography>
        }),
        columnHelper.accessor('reason_text', {
            header: () => <TableHeadingCell name={'Custom Reason'} />,
            cell: (info) => <Typography>{info.getValue()}</Typography>
        }),
        columnHelper.accessor('created_on', {
            header: () => <TableHeadingCell name={'Created'} />,
            cell: (info) => <Typography>{renderDateTime(info.getValue())}</Typography>
        }),
        columnHelper.accessor('updated_on', {
            header: () => <TableHeadingCell name={'Last Active'} />,
            cell: (info) => <Typography>{renderDateTime(info.getValue())}</Typography>
        })
    ];

    const table = useReactTable({
        data: appeals.data,
        columns: columns,
        getCoreRowModel: getCoreRowModel(),
        manualPagination: true,
        autoResetPageIndex: true
    });

    return <DataTable table={table} isLoading={isLoading} />;
};
