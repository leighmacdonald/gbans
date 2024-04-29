import AddIcon from '@mui/icons-material/Add';
import GavelIcon from '@mui/icons-material/Gavel';
import Button from '@mui/material/Button';
import Grid from '@mui/material/Unstable_Grid2';
import { createFileRoute } from '@tanstack/react-router';
import { intervalToDuration } from 'date-fns';
import { z } from 'zod';
import { AppealState } from '../api';
import { ContainerWithHeaderAndButtons } from '../component/ContainerWithHeaderAndButtons.tsx';
import { commonTableSearchSchema } from '../util/table.ts';

export const isPermanentBan = (start: Date, end: Date): boolean => {
    const dur = intervalToDuration({
        start,
        end
    });
    const { years } = dur;
    return years != null && years > 5;
};

const banSteamSearchSchema = z.object({
    ...commonTableSearchSchema,
    sortColumn: z.enum(['ban_id', 'source_id', 'target_id', 'deleted', 'reason', 'appeal_state', 'valid_until']).catch('ban_id'),
    source_id: z.string().catch(''),
    target_id: z.string().catch(''),
    appeal_state: z.nativeEnum(AppealState).catch(AppealState.Any),
    deleted: z.boolean().catch(false)
});

export const Route = createFileRoute('/_mod/admin/ban/steam')({
    component: AdminBanSteam,
    validateSearch: (search) => banSteamSearchSchema.parse(search)
});

function AdminBanSteam() {
    // const { page, rows, sortOrder, sortColumn, deleted, target_id, appeal_state, source_id } = Route.useSearch();
    // const [newSteamBans, setNewSteamBans] = useState<SteamBanRecord[]>([]);
    // const { sendFlash } = useUserFlashCtx();
    //
    // const onNewBanSteam = useCallback(async () => {
    //     try {
    //         const ban = await NiceModal.show<SteamBanRecord>(ModalBanSteam, {});
    //         setNewSteamBans((prevState) => {
    //             return [ban, ...prevState];
    //         });
    //         sendFlash('success', `Created steam ban successfully #${ban.ban_id}`);
    //     } catch (e) {
    //         logErr(e);
    //     }
    // }, [sendFlash]);

    // const onUnbanSteam = useCallback(
    //     async (ban: SteamBanRecord) => {
    //         try {
    //             await NiceModal.show(ModalUnbanSteam, {
    //                 banId: ban.ban_id,
    //                 personaName: ban.target_personaname
    //             });
    //             sendFlash('success', 'Unbanned successfully');
    //         } catch (e) {
    //             sendFlash('error', `Failed to unban: ${e}`);
    //         }
    //     },
    //     [sendFlash]
    // );
    //
    // const onEditSteam = useCallback(
    //     async (ban: SteamBanRecord) => {
    //         try {
    //             await NiceModal.show(ModalBanSteam, {
    //                 banId: ban.ban_id,
    //                 personaName: ban.target_personaname,
    //                 existing: ban
    //             });
    //             sendFlash('success', 'Updated ban successfully');
    //         } catch (e) {
    //             sendFlash('error', `Failed to update ban: ${e}`);
    //         }
    //     },
    //     [sendFlash]
    // );

    // const { data, count } = useBansSteam({
    //     limit: Number(rows ?? RowsPerPage.Ten),
    //     offset: Number((page ?? 0) * (rows ?? RowsPerPage.Ten)),
    //     order_by: sortColumn ?? 'ban_id',
    //     desc: (sortOrder ?? 'desc') == 'desc',
    //     source_id: source_id ?? '',
    //     target_id: target_id ?? '',
    //     appeal_state: Number(appeal_state ?? AppealState.Any),
    //     deleted: deleted ?? false
    // });

    // const allBans = useMemo(() => {
    //     if (newSteamBans.length > 0) {
    //         return [...newSteamBans, ...data];
    //     }
    //
    //     return data;
    // }, [data, newSteamBans]);

    // const onSubmit = useCallback(() => {
    //     // (values: SteamBanFilterValues) => {
    //     //     const newState = {
    //     //         appealState: values.appeal_state != AppealState.Any ? values.appeal_state : undefined,
    //     //         source: values.source_id != '' ? values.source_id : undefined,
    //     //         target: values.target_id != '' ? values.target_id : undefined,
    //     //         deleted: values.deleted ? true : undefined
    //     //     };
    //     //     setState(newState);
    // }, []);
    //
    // const onReset = useCallback(async () => {
    //     // setState({
    //     //     appealState: undefined,
    //     //     source: undefined,
    //     //     target: undefined,
    //     //     deleted: undefined
    //     // });
    //     // await formikHelpers.setFieldValue('source_id', '');
    //     // await formikHelpers.setFieldValue('target_id', '');
    // }, []);

    return (
        <Grid container>
            <Grid xs={12}>
                <ContainerWithHeaderAndButtons
                    title={'Steam Ban History'}
                    marginTop={0}
                    iconLeft={<GavelIcon />}
                    buttons={[
                        <Button
                            key={`ban-steam`}
                            variant={'contained'}
                            color={'success'}
                            startIcon={<AddIcon />}
                            sx={{ marginRight: 2 }}
                            // onClick={onNewBanSteam}
                        >
                            Create
                        </Button>
                    ]}
                >
                    {/*<Formik onReset={onReset} onSubmit={onSubmit} initialValues={{}}>*/}
                    <Grid container spacing={3}>
                        <Grid xs={12}>
                            <Grid container spacing={2}>
                                {/*<Grid xs={4} sm={3} md={2}>*/}
                                {/*    <SourceIDField />*/}
                                {/*</Grid>*/}
                                {/*<Grid xs={4} sm={3} md={2}>*/}
                                {/*    <TargetIDField />*/}
                                {/*</Grid>*/}
                                {/*<Grid xs={4} sm={3} md={2}>*/}
                                {/*    <AppealStateField />*/}
                                {/*</Grid>*/}
                                {/*<Grid xs={4} sm={3} md={2}>*/}
                                {/*    <DeletedField />*/}
                                {/*</Grid>*/}
                                {/*<Grid xs={4} sm={3} md={2}>*/}
                                {/*    <FilterButtons />*/}
                                {/*</Grid>*/}
                            </Grid>
                        </Grid>
                        <Grid xs={12}>
                            {/*<LazyTable<SteamBanRecord>*/}
                            {/*    showPager={true}*/}
                            {/*    count={count}*/}
                            {/*    rows={allBans}*/}
                            {/*    page={Number(state.page ?? 0)}*/}
                            {/*    rowsPerPage={Number(state.rows ?? RowsPerPage.Ten)}*/}
                            {/*    sortOrder={state.sortOrder}*/}
                            {/*    sortColumn={state.sortColumn}*/}
                            {/*    onSortColumnChanged={async (column) => {*/}
                            {/*        setState({ sortColumn: column });*/}
                            {/*    }}*/}
                            {/*    onSortOrderChanged={async (direction) => {*/}
                            {/*        setState({ sortOrder: direction });*/}
                            {/*    }}*/}
                            {/*    onPageChange={(_, newPage: number) => {*/}
                            {/*        setState({ page: newPage });*/}
                            {/*    }}*/}
                            {/*    onRowsPerPageChange={(event: ChangeEvent<HTMLInputElement | HTMLTextAreaElement>) => {*/}
                            {/*        setState({*/}
                            {/*            rows: Number(event.target.value),*/}
                            {/*            page: 0*/}
                            {/*        });*/}
                            {/*    }}*/}
                            {/*    columns={[*/}
                            {/*        {*/}
                            {/*            label: '#',*/}
                            {/*            tooltip: 'Ban ID',*/}
                            {/*            sortKey: 'ban_id',*/}
                            {/*            sortable: true,*/}
                            {/*            align: 'left',*/}
                            {/*            renderer: (row) => (*/}
                            {/*                <TableCellLink label={`#${row.ban_id.toString()}`} to={`/ban/${row.ban_id}`} />*/}
                            {/*            )*/}
                            {/*        },*/}
                            {/*        {*/}
                            {/*            label: 'A',*/}
                            {/*            tooltip: 'Ban Author',*/}
                            {/*            sortKey: 'source_personaname',*/}
                            {/*            sortable: true,*/}
                            {/*            align: 'center',*/}
                            {/*            renderer: (row) => (*/}
                            {/*                <SteamIDSelectField*/}
                            {/*                    steam_id={row.source_id}*/}
                            {/*                    personaname={row.source_personaname}*/}
                            {/*                    avatarhash={row.source_avatarhash}*/}
                            {/*                    field_name={'source_id'}*/}
                            {/*                />*/}
                            {/*            )*/}
                            {/*        },*/}
                            {/*        {*/}
                            {/*            label: 'Target',*/}
                            {/*            tooltip: 'Steam Name',*/}
                            {/*            sortKey: 'target_personaname',*/}
                            {/*            sortable: true,*/}
                            {/*            align: 'left',*/}
                            {/*            renderer: (row) => (*/}
                            {/*                <SteamIDSelectField*/}
                            {/*                    steam_id={row.target_id}*/}
                            {/*                    personaname={row.target_personaname}*/}
                            {/*                    avatarhash={row.target_avatarhash}*/}
                            {/*                    field_name={'target_id'}*/}
                            {/*                />*/}
                            {/*            )*/}
                            {/*        },*/}
                            {/*        {*/}
                            {/*            label: 'Reason',*/}
                            {/*            tooltip: 'Reason',*/}
                            {/*            sortKey: 'reason',*/}
                            {/*            sortable: true,*/}
                            {/*            align: 'left',*/}
                            {/*            renderer: (row) => (*/}
                            {/*                <Box>*/}
                            {/*                    <Tooltip*/}
                            {/*                        title={row.reason == BanReason.Custom ? row.reason_text : BanReason[row.reason]}*/}
                            {/*                    >*/}
                            {/*                        <Typography variant={'body1'}>{`${BanReason[row.reason]}`}</Typography>*/}
                            {/*                    </Tooltip>*/}
                            {/*                </Box>*/}
                            {/*            )*/}
                            {/*        },*/}
                            {/*        {*/}
                            {/*            label: 'Created',*/}
                            {/*            tooltip: 'Created On',*/}
                            {/*            sortType: 'date',*/}
                            {/*            align: 'left',*/}
                            {/*            width: '150px',*/}
                            {/*            virtual: true,*/}
                            {/*            virtualKey: 'created_on',*/}
                            {/*            renderer: (obj) => {*/}
                            {/*                return <Typography variant={'body1'}>{renderDate(obj.created_on)}</Typography>;*/}
                            {/*            }*/}
                            {/*        },*/}
                            {/*        {*/}
                            {/*            label: 'Expires',*/}
                            {/*            tooltip: 'Valid Until',*/}
                            {/*            sortType: 'date',*/}
                            {/*            align: 'left',*/}
                            {/*            width: '150px',*/}
                            {/*            virtual: true,*/}
                            {/*            virtualKey: 'valid_until',*/}
                            {/*            sortable: true,*/}
                            {/*            renderer: (obj) => {*/}
                            {/*                return <TableCellRelativeDateField date={obj.valid_until} />;*/}
                            {/*            }*/}
                            {/*        },*/}
                            {/*        {*/}
                            {/*            label: 'Duration',*/}
                            {/*            tooltip: 'Total Ban Duration',*/}
                            {/*            sortType: 'number',*/}
                            {/*            align: 'left',*/}
                            {/*            width: '150px',*/}
                            {/*            virtual: true,*/}
                            {/*            virtualKey: 'duration',*/}
                            {/*            renderer: (row) => {*/}
                            {/*                return isPermanentBan(row.created_on, row.valid_until) ? (*/}
                            {/*                    'Permanent'*/}
                            {/*                ) : (*/}
                            {/*                    <TableCellRelativeDateField date={row.created_on} compareDate={row.valid_until} />*/}
                            {/*                );*/}
                            {/*            }*/}
                            {/*        },*/}
                            {/*        {*/}
                            {/*            label: 'F',*/}
                            {/*            tooltip: 'Are friends also included in the ban',*/}
                            {/*            align: 'center',*/}
                            {/*            width: '50px',*/}
                            {/*            sortKey: 'include_friends',*/}
                            {/*            renderer: (row) => <TableCellBool enabled={row.include_friends} />*/}
                            {/*        },*/}
                            {/*        {*/}
                            {/*            label: 'E',*/}
                            {/*            tooltip:*/}
                            {/*                'Are othere players allowed to play from the same ip when a ban on that ip is active (eg. banned roomate/family)',*/}
                            {/*            align: 'center',*/}
                            {/*            width: '50px',*/}
                            {/*            sortKey: 'evade_ok',*/}
                            {/*            renderer: (row) => <TableCellBool enabled={row.evade_ok} />*/}
                            {/*        },*/}
                            {/*        {*/}
                            {/*            label: 'A',*/}
                            {/*            tooltip: 'Is this ban active (not deleted/inactive/unbanned)',*/}
                            {/*            align: 'center',*/}
                            {/*            width: '50px',*/}
                            {/*            sortKey: 'deleted',*/}
                            {/*            renderer: (row) => <TableCellBool enabled={!row.deleted} />*/}
                            {/*        },*/}
                            {/*        {*/}
                            {/*            label: 'Rep.',*/}
                            {/*            tooltip: 'Report',*/}
                            {/*            sortable: false,*/}
                            {/*            align: 'center',*/}
                            {/*            width: '20px',*/}
                            {/*            renderer: (row) =>*/}
                            {/*                row.report_id > 0 ? (*/}
                            {/*                    <Tooltip title={'View Report'}>*/}
                            {/*                        <>*/}
                            {/*                            <TableCellLink label={`#${row.report_id}`} to={`/report/${row.report_id}`} />*/}
                            {/*                        </>*/}
                            {/*                    </Tooltip>*/}
                            {/*                ) : (*/}
                            {/*                    <></>*/}
                            {/*                )*/}
                            {/*        },*/}
                            {/*        {*/}
                            {/*            label: 'Act.',*/}
                            {/*            tooltip: 'Actions',*/}
                            {/*            sortKey: 'reason',*/}
                            {/*            sortable: false,*/}
                            {/*            align: 'center',*/}
                            {/*            renderer: (row) => (*/}
                            {/*                <ButtonGroup fullWidth>*/}
                            {/*                    <IconButton*/}
                            {/*                        color={'warning'}*/}
                            {/*                        onClick={async () => {*/}
                            {/*                            await onEditSteam(row);*/}
                            {/*                        }}*/}
                            {/*                    >*/}
                            {/*                        <Tooltip title={'Edit Ban'}>*/}
                            {/*                            <EditIcon />*/}
                            {/*                        </Tooltip>*/}
                            {/*                    </IconButton>*/}
                            {/*                    <IconButton*/}
                            {/*                        color={'success'}*/}
                            {/*                        onClick={async () => {*/}
                            {/*                            await onUnbanSteam(row);*/}
                            {/*                        }}*/}
                            {/*                    >*/}
                            {/*                        <Tooltip title={'Remove Ban'}>*/}
                            {/*                            <UndoIcon />*/}
                            {/*                        </Tooltip>*/}
                            {/*                    </IconButton>*/}
                            {/*                </ButtonGroup>*/}
                            {/*            )*/}
                            {/*        }*/}
                            {/*    ]}*/}
                            {/*/>*/}
                        </Grid>
                    </Grid>
                    {/*</Formik>*/}
                </ContainerWithHeaderAndButtons>
            </Grid>
        </Grid>
    );
}
