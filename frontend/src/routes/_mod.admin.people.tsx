import FilterListIcon from '@mui/icons-material/FilterList';
import PersonSearchIcon from '@mui/icons-material/PersonSearch';
import VpnKeyIcon from '@mui/icons-material/VpnKey';
import IconButton from '@mui/material/IconButton';
import Stack from '@mui/material/Stack';
import Typography from '@mui/material/Typography';
import Grid from '@mui/material/Unstable_Grid2';
import { useQuery } from '@tanstack/react-query';
import { createFileRoute, useRouteContext } from '@tanstack/react-router';
import { createColumnHelper, getCoreRowModel, useReactTable } from '@tanstack/react-table';
import { fromUnixTime } from 'date-fns';
import { z } from 'zod';
import { apiSearchPeople, communityVisibilityState, PermissionLevel, permissionLevelString, Person } from '../api';
import { ContainerWithHeader } from '../component/ContainerWithHeader.tsx';
import { DataTable, HeadingCell } from '../component/DataTable.tsx';
import { Paginator } from '../component/Paginator.tsx';
import { PersonCell } from '../component/PersonCell.tsx';
import { commonTableSearchSchema, LazyResult } from '../util/table.ts';
import { renderDate, renderDateTime } from '../util/text.tsx';

const peopleSearchSchema = z.object({
    ...commonTableSearchSchema,
    sortColumn: z.enum(['vac_bans', 'steam_id', 'timecreated', 'personaname']).catch('timecreated'),
    steam_id: z.string().catch(''),
    personaname: z.string().catch('')
});

export const Route = createFileRoute('/_mod/admin/people')({
    component: AdminPeople,
    validateSearch: (search) => peopleSearchSchema.parse(search)
});

function AdminPeople() {
    const { hasPermission } = useRouteContext({ from: '/_mod/admin/people' });
    const { steam_id, page, personaname, sortColumn, rows, sortOrder } = Route.useSearch();

    const { data: people, isLoading } = useQuery({
        queryKey: ['people', {}],
        queryFn: async () => {
            return await apiSearchPeople({
                personaname: personaname,
                desc: sortOrder == 'desc',
                offset: Number((page ?? 0) * rows),
                limit: rows,
                order_by: sortColumn,
                target_id: steam_id,
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

    return (
        <Grid container spacing={2}>
            <Grid xs={12}>
                <ContainerWithHeader title={'Person Filters'} iconLeft={<FilterListIcon />}>
                    {/*<Formik*/}
                    {/*    onSubmit={onFilterSubmit}*/}
                    {/*    onReset={onFilterReset}*/}
                    {/*    initialValues={{*/}
                    {/*        personaname: '',*/}
                    {/*        target_id: '',*/}
                    {/*        ip: ''*/}
                    {/*    }}*/}
                    {/*    validationSchema={validationSchema}*/}
                    {/*>*/}
                    {/*<Grid container spacing={2}>*/}
                    {/*    <Grid xs={6} sm={4} md={3}>*/}
                    {/*        <TargetIDField />*/}
                    {/*    </Grid>*/}
                    {/*    <Grid xs={6} sm={4} md={3}>*/}
                    {/*        <PersonanameField />*/}
                    {/*    </Grid>*/}
                    {/*    <Grid xs={6} sm={4} md={3}>*/}
                    {/*        <IPField />*/}
                    {/*    </Grid>*/}
                    {/*    <Grid xs={6} sm={4} md={3}>*/}
                    {/*        <FilterButtons />*/}
                    {/*    </Grid>*/}
                    {/*</Grid>*/}
                    {/*</Formik>*/}
                </ContainerWithHeader>
            </Grid>
            <Grid xs={12}>
                <ContainerWithHeader title={'Player Search'} iconLeft={<PersonSearchIcon />}>
                    <PeopleTable
                        people={people ?? { data: [], count: 0 }}
                        isLoading={isLoading}
                        isAdmin={hasPermission(PermissionLevel.Admin)}
                    />
                    <Paginator page={page} rows={rows} />
                </ContainerWithHeader>
            </Grid>
        </Grid>
    );
}

const columnHelper = createColumnHelper<Person>();

const PeopleTable = ({ people, isLoading, isAdmin }: { people: LazyResult<Person>; isLoading: boolean; isAdmin: boolean }) => {
    const columns = [
        columnHelper.accessor('steam_id', {
            header: () => <HeadingCell name={'View'} />,
            cell: (info) => (
                <PersonCell
                    steam_id={people.data[info.row.index].steam_id}
                    personaname={people.data[info.row.index].personaname}
                    avatar_hash={people.data[info.row.index].avatarhash}
                />
            )
        }),
        columnHelper.accessor('communityvisibilitystate', {
            header: () => <HeadingCell name={'Profile'} />,
            cell: (info) => {
                return (
                    <Typography variant={'body1'}>{info.getValue() == communityVisibilityState.Public ? 'Public' : 'Private'}</Typography>
                );
            }
        }),
        columnHelper.accessor('vac_bans', {
            header: () => <HeadingCell name={'Reporter'} />,
            cell: (info) => <Typography variant={'body1'}>{info.getValue()}</Typography>
        }),
        columnHelper.accessor('community_banned', {
            header: () => <HeadingCell name={'Subject'} />,
            cell: (info) => <Typography variant={'body1'}>{info.getValue() ? 'Yes' : 'No'}</Typography>
        }),
        columnHelper.accessor('timecreated', {
            header: () => <HeadingCell name={'Reason'} />,
            cell: (info) => <Typography>{renderDate(fromUnixTime(info.getValue()))}</Typography>
        }),
        columnHelper.accessor('created_on', {
            header: () => <HeadingCell name={'Created'} />,
            cell: (info) => <Typography>{renderDateTime(info.getValue())}</Typography>
        }),
        columnHelper.accessor('permission_level', {
            header: () => <HeadingCell name={'Updated'} />,
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
