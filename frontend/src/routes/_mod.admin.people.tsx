import FilterListIcon from '@mui/icons-material/FilterList';
import PersonSearchIcon from '@mui/icons-material/PersonSearch';
import Grid from '@mui/material/Unstable_Grid2';
import { createFileRoute } from '@tanstack/react-router';
import { z } from 'zod';
import { ContainerWithHeader } from '../component/ContainerWithHeader.tsx';
import { commonTableSearchSchema } from '../util/table.ts';

const peopleSearchSchema = z.object({
    ...commonTableSearchSchema,
    // sortColumn: z.enum(['ban_asn_id', 'source_id', 'target_id', 'deleted', 'reason', 'as_num', 'valid_until']).catch('ban_asn_id'),
    steam_id: z.string().catch(''),
    personaname: z.string().catch('')
});

export const Route = createFileRoute('/_mod/admin/people')({
    component: AdminPeople,
    validateSearch: (search) => peopleSearchSchema.parse(search)
});

function AdminPeople() {
    // const { hasPermission } = useRouteContext({ from: '/_mod/admin/people' });
    //
    // const { data, count, loading } = usePeople({
    //     personaname: state.personaname,
    //     deleted: false,
    //     desc: state.sortOrder == 'desc',
    //     limit: Number(state.rows ?? RowsPerPage.Ten),
    //     offset: Number((state.page ?? 0) * (state.rows ?? RowsPerPage.Ten)),
    //     order_by: state.sortColumn ?? 'created_on',
    //     target_id: state.target_id,
    //     ip: state.ip
    // });
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
                    {/*<LazyTable*/}
                    {/*    count={count}*/}
                    {/*    sortOrder={state.sortOrder}*/}
                    {/*    sortColumn={state.sortColumn}*/}
                    {/*    onSortColumnChanged={async (column) => {*/}
                    {/*        setState({ sortColumn: column });*/}
                    {/*    }}*/}
                    {/*    onSortOrderChanged={async (direction) => {*/}
                    {/*        setState({ sortOrder: direction });*/}
                    {/*    }}*/}
                    {/*    onRowsPerPageChange={(event: ChangeEvent<HTMLInputElement | HTMLTextAreaElement>) => {*/}
                    {/*        setState({*/}
                    {/*            rows: Number(event.target.value),*/}
                    {/*            page: 0*/}
                    {/*        });*/}
                    {/*    }}*/}
                    {/*    onPageChange={(_, newPage) => {*/}
                    {/*        setState({ page: newPage });*/}
                    {/*    }}*/}
                    {/*    rows={data}*/}
                    {/*    showPager*/}
                    {/*    page={Number(state.page ?? 0)}*/}
                    {/*    rowsPerPage={Number(state.rows ?? RowsPerPage.TwentyFive)}*/}
                    {/*    columns={[*/}
                    {/*        {*/}
                    {/*            label: 'Steam ID',*/}
                    {/*            tooltip: 'Steam ID',*/}
                    {/*            sortKey: 'steam_id',*/}
                    {/*            align: 'left',*/}
                    {/*            sortable: true,*/}
                    {/*            renderer: (row) => (*/}
                    {/*                <PersonCell*/}
                    {/*                    steam_id={row.steam_id}*/}
                    {/*                    personaname={row.personaname != '' ? row.personaname : row.steam_id}*/}
                    {/*                    avatar_hash={row.avatarhash != '' ? row.avatarhash : defaultAvatarHash}*/}
                    {/*                />*/}
                    {/*            )*/}
                    {/*        },*/}
                    {/*        {*/}
                    {/*            label: 'Profile',*/}
                    {/*            tooltip: 'Community Visibility State',*/}
                    {/*            sortKey: 'communityvisibilitystate',*/}
                    {/*            align: 'left',*/}
                    {/*            sortable: true,*/}
                    {/*            renderer: (row) => (*/}
                    {/*                <Typography variant={'body1'}>*/}
                    {/*                    {row.communityvisibilitystate == communityVisibilityState.Public ? 'Public' : 'Private'}*/}
                    {/*                </Typography>*/}
                    {/*            )*/}
                    {/*        },*/}
                    {/*        {*/}
                    {/*            label: 'Vac Ban',*/}
                    {/*            tooltip: 'Amount of vac bans',*/}
                    {/*            sortKey: 'vac_bans',*/}
                    {/*            align: 'left',*/}
                    {/*            sortable: true,*/}
                    {/*            renderer: (row) => <Typography variant={'body1'}>{row.vac_bans}</Typography>*/}
                    {/*        },*/}
                    {/*        {*/}
                    {/*            label: 'Comm. Ban',*/}
                    {/*            tooltip: 'Is the player community banned',*/}
                    {/*            sortKey: 'community_banned',*/}
                    {/*            align: 'left',*/}
                    {/*            sortable: true,*/}
                    {/*            renderer: (row) => <Typography variant={'body1'}>{row.community_banned ? 'Yes' : 'No'}</Typography>*/}
                    {/*        },*/}
                    {/*        {*/}
                    {/*            label: 'Account Created',*/}
                    {/*            tooltip: 'When the account was created',*/}
                    {/*            sortKey: 'timecreated',*/}
                    {/*            align: 'left',*/}
                    {/*            sortable: true,*/}
                    {/*            renderer: (row) => (*/}
                    {/*                <Typography variant={'body1'}>*/}
                    {/*                    {!isValidSteamDate(fromUnixTime(row.timecreated))*/}
                    {/*                        ? 'Unknown'*/}
                    {/*                        : renderDate(fromUnixTime(row.timecreated))}*/}
                    {/*                </Typography>*/}
                    {/*            )*/}
                    {/*        },*/}
                    {/*        {*/}
                    {/*            label: 'First Seen',*/}
                    {/*            tooltip: 'When the user was first seen',*/}
                    {/*            sortable: true,*/}
                    {/*            sortKey: 'created_on',*/}
                    {/*            align: 'left',*/}
                    {/*            width: '150px',*/}
                    {/*            renderer: (obj) => {*/}
                    {/*                return <Typography variant={'body1'}>{renderDate(obj.created_on)}</Typography>;*/}
                    {/*            }*/}
                    {/*        },*/}
                    {/*        {*/}
                    {/*            label: 'Perms',*/}
                    {/*            tooltip: 'Permission Level',*/}
                    {/*            sortKey: 'permission_level',*/}
                    {/*            align: 'left',*/}
                    {/*            sortable: true,*/}
                    {/*            renderer: (row) => <Typography variant={'body1'}>{permissionLevelString(row.permission_level)}</Typography>*/}
                    {/*        },*/}
                    {/*        {*/}
                    {/*            virtual: true,*/}
                    {/*            virtualKey: 'actions',*/}
                    {/*            label: '',*/}
                    {/*            tooltip: '',*/}
                    {/*            align: 'right',*/}
                    {/*            renderer: (obj) => {*/}
                    {/*                return (*/}
                    {/*                    <ButtonGroup>*/}
                    {/*                        <IconButton*/}
                    {/*                            disabled={!auth.user || auth.user.permission_level < PermissionLevel.Admin}*/}
                    {/*                            color={'warning'}*/}
                    {/*                            onClick={async () => {*/}
                    {/*                                try {*/}
                    {/*                                    await onEditPerson(obj);*/}
                    {/*                                } catch (e) {*/}
                    {/*                                    logErr(e);*/}
                    {/*                                }*/}
                    {/*                            }}*/}
                    {/*                        >*/}
                    {/*                            <VpnKeyIcon />*/}
                    {/*                        </IconButton>*/}
                    {/*                    </ButtonGroup>*/}
                    {/*                );*/}
                    {/*            }*/}
                    {/*        }*/}
                    {/*    ]}*/}
                    {/*/>*/}
                </ContainerWithHeader>
            </Grid>
        </Grid>
    );
}
