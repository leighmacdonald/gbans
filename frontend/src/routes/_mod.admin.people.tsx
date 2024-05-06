import FilterListIcon from '@mui/icons-material/FilterList';
import PersonSearchIcon from '@mui/icons-material/PersonSearch';
import VpnKeyIcon from '@mui/icons-material/VpnKey';
import IconButton from '@mui/material/IconButton';
import Stack from '@mui/material/Stack';
import Typography from '@mui/material/Typography';
import Grid from '@mui/material/Unstable_Grid2';
import { useForm } from '@tanstack/react-form';
import { useQuery } from '@tanstack/react-query';
import { createFileRoute, useNavigate, useRouteContext } from '@tanstack/react-router';
import { createColumnHelper, getCoreRowModel, useReactTable } from '@tanstack/react-table';
import { zodValidator } from '@tanstack/zod-form-adapter';
import { fromUnixTime } from 'date-fns';
import { z } from 'zod';
import { apiSearchPeople, communityVisibilityState, PermissionLevel, permissionLevelString, Person } from '../api';
import { ContainerWithHeader } from '../component/ContainerWithHeader.tsx';
import { DataTable } from '../component/DataTable.tsx';
import { Paginator } from '../component/Paginator.tsx';
import { PersonCell } from '../component/PersonCell.tsx';
import { TableHeadingCell } from '../component/TableHeadingCell.tsx';
import { Buttons } from '../component/field/Buttons.tsx';
import { makeSteamidValidatorsOptional } from '../component/field/SteamIDField.tsx';
import { TextFieldSimple } from '../component/field/TextFieldSimple.tsx';
import { commonTableSearchSchema, LazyResult, RowsPerPage } from '../util/table.ts';
import { renderDate, renderDateTime } from '../util/text.tsx';

const peopleSearchSchema = z.object({
    ...commonTableSearchSchema,
    sortColumn: z.enum(['vac_bans', 'steam_id', 'timecreated', 'personaname']).optional(),
    steam_id: z.string().optional(),
    personaname: z.string().optional()
});

export const Route = createFileRoute('/_mod/admin/people')({
    component: AdminPeople,
    validateSearch: (search) => peopleSearchSchema.parse(search)
});

function AdminPeople() {
    const defaultRows = RowsPerPage.TwentyFive;
    const navigate = useNavigate({ from: Route.fullPath });
    const { hasPermission } = useRouteContext({ from: '/_mod/admin/people' });
    const { steam_id, page, personaname, sortColumn, rows, sortOrder } = Route.useSearch();
    const { data: people, isLoading } = useQuery({
        queryKey: ['people', { rows, page, sortColumn, sortOrder, personaname, steam_id }],
        queryFn: async () => {
            return await apiSearchPeople({
                personaname: personaname ?? '',
                desc: sortOrder == 'desc',
                offset: (page ?? 0) * (rows ?? defaultRows),
                limit: rows ?? defaultRows,
                order_by: sortColumn,
                target_id: steam_id ?? '',
                ip: ''
            });
        }
    });
    //
    // const onFilterSubmit = useCallback(
    //     (values: PeopleFilterValues) => {
    //         setState(values);
    //     },
    //     [setState]
    // );
    //
    // const onFilterReset = useCallback(() => {
    //     setState({
    //         ip: '',
    //         personaname: '',
    //         target_id: ''
    //     });
    // }, [setState]);
    //
    // const onEditPerson = useCallback(async (person: Person) => {
    //     try {
    //         await NiceModal.show<Person>(ModalPersonEditor, {
    //             person
    //         });
    //     } catch (e) {
    //         logErr(e);
    //     }
    // }, []);

    const { Field, Subscribe, handleSubmit, reset } = useForm({
        onSubmit: async ({ value }) => {
            await navigate({ to: '/admin/people', search: (prev) => ({ ...prev, ...value }) });
        },
        validatorAdapter: zodValidator,
        validators: {
            onChange: peopleSearchSchema
        },
        defaultValues: {
            steam_id: steam_id ?? '',
            personaname: personaname ?? ''
        }
    });

    const clear = async () => {
        await navigate({
            to: '/admin/people',
            search: (prev) => ({ ...prev, steam_id: undefined, personaname: undefined })
        });
    };

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
                            <Grid xs={6} md={6}>
                                <Field
                                    name={'steam_id'}
                                    validators={makeSteamidValidatorsOptional()}
                                    children={(props) => {
                                        return <TextFieldSimple {...props} label={'Steam ID'} fullwidth={true} />;
                                    }}
                                />
                            </Grid>

                            <Grid xs={6} md={6}>
                                <Field
                                    name={'personaname'}
                                    children={(props) => {
                                        return <TextFieldSimple {...props} label={'Name'} fullwidth={true} />;
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
                <ContainerWithHeader title={'Player Search'} iconLeft={<PersonSearchIcon />}>
                    <PeopleTable
                        people={people ?? { data: [], count: 0 }}
                        isLoading={isLoading}
                        isAdmin={hasPermission(PermissionLevel.Admin)}
                    />
                    <Paginator page={page ?? 0} rows={rows ?? defaultRows} data={people} path={'/admin/people'} />
                </ContainerWithHeader>
            </Grid>
        </Grid>
    );
}

const columnHelper = createColumnHelper<Person>();

const PeopleTable = ({ people, isLoading, isAdmin }: { people: LazyResult<Person>; isLoading: boolean; isAdmin: boolean }) => {
    const columns = [
        columnHelper.accessor('steam_id', {
            header: () => <TableHeadingCell name={'View'} />,
            cell: (info) => (
                <PersonCell
                    steam_id={people.data[info.row.index].steam_id}
                    personaname={people.data[info.row.index].personaname}
                    avatar_hash={people.data[info.row.index].avatarhash}
                />
            )
        }),
        columnHelper.accessor('communityvisibilitystate', {
            header: () => <TableHeadingCell name={'Profile'} />,
            cell: (info) => {
                return (
                    <Typography variant={'body1'}>{info.getValue() == communityVisibilityState.Public ? 'Public' : 'Private'}</Typography>
                );
            }
        }),
        columnHelper.accessor('vac_bans', {
            header: () => <TableHeadingCell name={'Reporter'} />,
            cell: (info) => <Typography variant={'body1'}>{info.getValue()}</Typography>
        }),
        columnHelper.accessor('community_banned', {
            header: () => <TableHeadingCell name={'Subject'} />,
            cell: (info) => <Typography variant={'body1'}>{info.getValue() ? 'Yes' : 'No'}</Typography>
        }),
        columnHelper.accessor('timecreated', {
            header: () => <TableHeadingCell name={'Reason'} />,
            cell: (info) => <Typography>{renderDate(fromUnixTime(info.getValue()))}</Typography>
        }),
        columnHelper.accessor('created_on', {
            header: () => <TableHeadingCell name={'Created'} />,
            cell: (info) => <Typography>{renderDateTime(info.getValue())}</Typography>
        }),
        columnHelper.accessor('permission_level', {
            header: () => <TableHeadingCell name={'Updated'} />,
            cell: (info) => (
                <Stack direction={'row'}>
                    {isAdmin && (
                        <IconButton color={'warning'}>
                            <VpnKeyIcon />
                        </IconButton>
                    )}
                    <Typography>{permissionLevelString(info.getValue())}</Typography>
                </Stack>
            )
        })
    ];

    const table = useReactTable({
        data: people.data,
        columns: columns,
        getCoreRowModel: getCoreRowModel(),
        manualPagination: true,
        autoResetPageIndex: true
    });

    return <DataTable table={table} isLoading={isLoading} />;
};
