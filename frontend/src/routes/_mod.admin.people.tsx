import { useMemo } from 'react';
import NiceModal from '@ebay/nice-modal-react';
import FilterListIcon from '@mui/icons-material/FilterList';
import PersonSearchIcon from '@mui/icons-material/PersonSearch';
import VpnKeyIcon from '@mui/icons-material/VpnKey';
import IconButton from '@mui/material/IconButton';
import Typography from '@mui/material/Typography';
import Grid from '@mui/material/Unstable_Grid2';
import { useForm } from '@tanstack/react-form';
import { useQuery } from '@tanstack/react-query';
import { createFileRoute, useNavigate, useRouteContext } from '@tanstack/react-router';
import { ColumnDef, getCoreRowModel, useReactTable } from '@tanstack/react-table';
import { fromUnixTime } from 'date-fns';
import { z } from 'zod';
import { apiSearchPeople, communityVisibilityState, PermissionLevel, permissionLevelString, Person } from '../api';
import { ContainerWithHeader } from '../component/ContainerWithHeader.tsx';
import { DataTable } from '../component/DataTable.tsx';
import { Paginator } from '../component/Paginator.tsx';
import { PersonCell } from '../component/PersonCell.tsx';
import { Title } from '../component/Title';
import { Buttons } from '../component/field/Buttons.tsx';
import { CheckboxSimple } from '../component/field/CheckboxSimple.tsx';
import { SteamIDField } from '../component/field/SteamIDField.tsx';
import { TextFieldSimple } from '../component/field/TextFieldSimple.tsx';
import { ModalPersonEditor } from '../component/modal';
import { useUserFlashCtx } from '../hooks/useUserFlashCtx.ts';
import { commonTableSearchSchema, LazyResult, RowsPerPage } from '../util/table.ts';
import { renderDate, renderDateTime } from '../util/time.ts';

const peopleSearchSchema = z.object({
    ...commonTableSearchSchema,
    sortColumn: z.enum(['vac_bans', 'steam_id', 'timecreated', 'personaname', 'created_on']).optional(),
    steam_id: z.string().catch(''),
    personaname: z.string().catch(''),
    staff_only: z.boolean().catch(false)
});

export const Route = createFileRoute('/_mod/admin/people')({
    component: AdminPeople,
    validateSearch: (search) => peopleSearchSchema.parse(search)
});

function AdminPeople() {
    const { sendFlash } = useUserFlashCtx();
    const defaultRows = RowsPerPage.TwentyFive;
    const navigate = useNavigate({ from: Route.fullPath });
    const { hasPermission } = useRouteContext({ from: '/_mod/admin/people' });
    const { steam_id, staff_only, pageIndex, personaname, sortColumn, pageSize, sortOrder } = Route.useSearch();
    const { data: people, isLoading } = useQuery({
        queryKey: ['people', { pageSize, pageIndex, sortColumn, sortOrder, personaname, steam_id }],
        queryFn: async () => {
            return await apiSearchPeople({
                personaname: personaname ?? '',
                desc: (sortOrder ?? 'desc') == 'desc',
                offset: (pageIndex ?? 0) * (pageSize ?? defaultRows),
                limit: pageSize ?? defaultRows,
                staff_only: staff_only ?? false,
                order_by: sortColumn ?? 'created_on',
                target_id: steam_id ?? '',
                ip: ''
            });
        }
    });

    const onEditPerson = async (person: Person) => {
        try {
            await NiceModal.show<Person>(ModalPersonEditor, {
                person
            });
            sendFlash('success', 'Updated permission level successfully');
        } catch (e) {
            sendFlash('error', `${e}`);
        }
    };

    const { Field, Subscribe, handleSubmit, reset } = useForm({
        onSubmit: async ({ value }) => {
            await navigate({ to: '/admin/people', search: (prev) => ({ ...prev, ...value }) });
        },
        validators: {
            onChange: z.object({
                steam_id: z.string(),
                personaname: z.string(),
                staff_only: z.boolean()
            })
        },
        defaultValues: {
            steam_id: steam_id ?? '',
            personaname: personaname ?? '',
            staff_only: staff_only ?? false
        }
    });

    const clear = async () => {
        await navigate({
            to: '/admin/people',
            search: (prev) => ({ ...prev, steam_id: '', personaname: '' })
        });
    };

    return (
        <Grid container spacing={2}>
            <Title>People</Title>
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
                                    name={'steam_id'}
                                    children={(props) => {
                                        return <SteamIDField {...props} label={'Steam ID'} fullwidth={true} />;
                                    }}
                                />
                            </Grid>

                            <Grid xs={6} md={4}>
                                <Field
                                    name={'personaname'}
                                    children={(props) => {
                                        return <TextFieldSimple {...props} label={'Name'} fullwidth={true} />;
                                    }}
                                />
                            </Grid>

                            <Grid xs={6} md={4}>
                                <Field
                                    name={'staff_only'}
                                    children={(props) => {
                                        return <CheckboxSimple {...props} label={'Staff Only'} fullwidth={true} />;
                                    }}
                                />
                            </Grid>

                            <Grid xs={12} mdOffset="auto">
                                <Subscribe
                                    selector={(state) => [state.canSubmit, state.isSubmitting]}
                                    children={([canSubmit, isSubmitting]) => (
                                        <Buttons
                                            reset={reset}
                                            canSubmit={canSubmit}
                                            isSubmitting={isSubmitting}
                                            onClear={clear}
                                        />
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
                        onEditPerson={onEditPerson}
                    />
                    <Paginator
                        page={pageIndex ?? 0}
                        rows={pageSize ?? defaultRows}
                        data={people}
                        path={'/admin/people'}
                    />
                </ContainerWithHeader>
            </Grid>
        </Grid>
    );
}

const PeopleTable = ({
    people,
    isLoading,
    isAdmin,
    onEditPerson
}: {
    people: LazyResult<Person>;
    isLoading: boolean;
    isAdmin: boolean;
    onEditPerson: (person: Person) => Promise<void>;
}) => {
    const columns = useMemo<ColumnDef<Person>[]>(
        () => [
            {
                accessorKey: 'source_id',
                header: 'Profile',
                cell: (info) => {
                    return typeof people.data[info.row.index] === 'undefined' ? (
                        ''
                    ) : (
                        <PersonCell
                            showCopy={true}
                            steam_id={people.data[info.row.index].steam_id}
                            personaname={people.data[info.row.index].personaname}
                            avatar_hash={people.data[info.row.index].avatarhash}
                        />
                    );
                }
            },
            {
                accessorKey: 'communityvisibilitystate',
                header: 'Visibility',
                size: 50,
                cell: (info) => (
                    <Typography variant={'body1'}>
                        {info.getValue() == communityVisibilityState.Public ? 'Public' : 'Private'}
                    </Typography>
                )
            },
            {
                accessorKey: 'vac_bans',
                header: 'Vac Ban',
                size: 50,
                cell: (info) => <Typography variant={'body1'}>{info.getValue() ? 'Yes' : 'No'}</Typography>
            },
            {
                accessorKey: 'community_banned',
                header: 'Comm. Ban',
                size: 50,
                cell: (info) => <Typography variant={'body1'}>{info.getValue() ? 'Yes' : 'No'}</Typography>
            },
            {
                accessorKey: 'timecreated',
                header: 'Account Created',
                size: 100,
                cell: (info) => <Typography>{renderDate(fromUnixTime(info.getValue() as number))}</Typography>
            },
            {
                accessorKey: 'created_on',
                header: 'First Seen',
                size: 100,
                cell: (info) => <Typography>{renderDateTime(info.getValue() as Date)}</Typography>
            },
            {
                accessorKey: 'permission_level',
                header: 'Perms',
                size: 80,
                cell: (info) => (
                    <Typography>
                        {permissionLevelString(
                            info.row.original
                                ? info.row.original.permission_level
                                : (PermissionLevel.Guest as PermissionLevel)
                        )}
                    </Typography>
                )
            },
            {
                id: 'actions',
                header: 'Edit',
                size: 30,
                cell: (info) => {
                    return isAdmin ? (
                        <IconButton color={'warning'} onClick={() => onEditPerson(info.row.original)}>
                            <VpnKeyIcon />
                        </IconButton>
                    ) : (
                        <></>
                    );
                }
            }
        ],
        [isAdmin, onEditPerson, people.data]
    );
    const table = useReactTable({
        data: people.data,
        columns: columns,
        getCoreRowModel: getCoreRowModel(),
        manualPagination: true,
        autoResetPageIndex: true
    });

    return <DataTable table={table} isLoading={isLoading} />;
};
