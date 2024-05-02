import AddIcon from '@mui/icons-material/Add';
import GavelIcon from '@mui/icons-material/Gavel';
import Button from '@mui/material/Button';
import Link from '@mui/material/Link';
import Typography from '@mui/material/Typography';
import Grid from '@mui/material/Unstable_Grid2';
import { useQuery } from '@tanstack/react-query';
import { createFileRoute, Link as RouterLink } from '@tanstack/react-router';
import { createColumnHelper, getCoreRowModel, useReactTable } from '@tanstack/react-table';
import { z } from 'zod';
import { apiGetBansSteam, AppealState, BanReason, BanReasons, SteamBanRecord } from '../api';
import { ContainerWithHeaderAndButtons } from '../component/ContainerWithHeaderAndButtons.tsx';
import { DataTable, HeadingCell } from '../component/DataTable.tsx';
import { PersonCell } from '../component/PersonCell.tsx';
import { TableCellBool } from '../component/table/TableCellBool.tsx';
import { TableCellRelativeDateField } from '../component/table/TableCellRelativeDateField.tsx';
import { commonTableSearchSchema, isPermanentBan, LazyResult } from '../util/table.ts';
import { renderDate } from '../util/text.tsx';

const banSteamSearchSchema = z.object({
    ...commonTableSearchSchema,
    sortColumn: z
        .enum(['ban_id', 'source_id', 'target_id', 'deleted', 'reason', 'created_on', 'valid_until', 'appeal_state'])
        .catch('ban_id'),
    source_id: z.string().catch(''),
    target_id: z.string().catch(''),
    reason: z.nativeEnum(BanReason).optional(),
    appeal_state: z.nativeEnum(AppealState).catch(AppealState.Any),
    deleted: z.boolean().catch(false)
});

export const Route = createFileRoute('/_mod/admin/ban/steam')({
    component: AdminBanSteam,
    validateSearch: (search) => banSteamSearchSchema.parse(search)
});

function AdminBanSteam() {
    const { page, rows, sortOrder, sortColumn, target_id, source_id } = Route.useSearch();
    const { data: bans, isLoading } = useQuery({
        queryKey: ['steamBans'],
        queryFn: async () => {
            return await apiGetBansSteam({
                limit: Number(rows),
                offset: Number((page ?? 0) * rows),
                order_by: sortColumn ?? 'ban_id',
                desc: sortOrder == 'desc',
                source_id: source_id,
                target_id: target_id
            });
        }
    });
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
                            <ReportTable bans={bans ?? { data: [], count: 0 }} isLoading={isLoading} />
                        </Grid>
                    </Grid>
                    {/*</Formik>*/}
                </ContainerWithHeaderAndButtons>
            </Grid>
        </Grid>
    );
}

const columnHelper = createColumnHelper<SteamBanRecord>();

const ReportTable = ({ bans, isLoading }: { bans: LazyResult<SteamBanRecord>; isLoading: boolean }) => {
    const columns = [
        columnHelper.accessor('ban_id', {
            header: () => <HeadingCell name={'Ban ID'} />,
            cell: (info) => (
                <Link component={RouterLink} to={`/ban/$ban_id`} params={{ ban_id: info.getValue() }}>
                    {`#${info.getValue()}`}
                </Link>
            )
        }),
        columnHelper.accessor('source_id', {
            header: () => <HeadingCell name={'Author'} />,
            cell: (info) => (
                <PersonCell
                    steam_id={bans.data[info.row.index].source_id}
                    personaname={bans.data[info.row.index].source_personaname}
                    avatar_hash={bans.data[info.row.index].source_avatarhash}
                />
            )
        }),
        columnHelper.accessor('target_id', {
            header: () => <HeadingCell name={'Subject'} />,
            cell: (info) => (
                <PersonCell
                    steam_id={bans.data[info.row.index].target_id}
                    personaname={bans.data[info.row.index].target_personaname}
                    avatar_hash={bans.data[info.row.index].target_avatarhash}
                />
            )
        }),
        columnHelper.accessor('reason', {
            header: () => <HeadingCell name={'Reason'} />,
            cell: (info) => <Typography>{BanReasons[info.getValue()]}</Typography>
        }),
        columnHelper.accessor('created_on', {
            header: () => <HeadingCell name={'Created'} />,
            cell: (info) => <Typography>{renderDate(info.getValue())}</Typography>
        }),
        columnHelper.accessor('valid_until', {
            header: () => <HeadingCell name={'Expires'} />,
            cell: (info) =>
                isPermanentBan(bans.data[info.row.index].created_on, bans.data[info.row.index].valid_until) ? (
                    'Permanent'
                ) : (
                    <TableCellRelativeDateField
                        date={bans.data[info.row.index].created_on}
                        compareDate={bans.data[info.row.index].valid_until}
                    />
                )
        }),
        columnHelper.accessor('include_friends', {
            header: () => <HeadingCell name={'F'} />,
            cell: (info) => <TableCellBool enabled={info.getValue()} />
        }),
        columnHelper.accessor('evade_ok', {
            header: () => <HeadingCell name={'E'} />,
            cell: (info) => <TableCellBool enabled={info.getValue()} />
        }),
        columnHelper.accessor('report_id', {
            header: () => <HeadingCell name={'Rep.'} />,
            cell: (info) =>
                info.getValue() > 0 && (
                    <Link component={RouterLink} to={`/report/$reportId`} params={{ reportId: info.getValue() }}>
                        {`#${info.getValue()}`}
                    </Link>
                )
        })
    ];

    const table = useReactTable({
        data: bans.data,
        columns: columns,
        getCoreRowModel: getCoreRowModel(),
        manualPagination: true,
        autoResetPageIndex: true
    });

    return <DataTable table={table} isLoading={isLoading} />;
};
