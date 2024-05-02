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
import { apiGetBansASN, ASNBanRecord, BanReasons } from '../api';
import { ContainerWithHeaderAndButtons } from '../component/ContainerWithHeaderAndButtons.tsx';
import { DataTable, HeadingCell } from '../component/DataTable.tsx';
import { PersonCell } from '../component/PersonCell.tsx';
import { TableCellRelativeDateField } from '../component/table/TableCellRelativeDateField.tsx';
import { commonTableSearchSchema, isPermanentBan, LazyResult } from '../util/table.ts';
import { renderDate } from '../util/text.tsx';

const banASNSearchSchema = z.object({
    ...commonTableSearchSchema,
    sortColumn: z.enum(['ban_asn_id', 'source_id', 'target_id', 'deleted', 'reason', 'as_num', 'valid_until']).catch('ban_asn_id'),
    source_id: z.string().catch(''),
    target_id: z.string().catch(''),
    as_num: z.number().catch(0),
    deleted: z.boolean().catch(false)
});

export const Route = createFileRoute('/_mod/admin/ban/asn')({
    component: AdminBanASN,
    validateSearch: (search) => banASNSearchSchema.parse(search)
});

function AdminBanASN() {
    const { page, rows, sortOrder, sortColumn, target_id, source_id } = Route.useSearch();
    const { data: bans, isLoading } = useQuery({
        queryKey: ['steamBans', { page, rows, sortOrder, sortColumn, target_id, source_id }],
        queryFn: async () => {
            return await apiGetBansASN({
                limit: Number(rows),
                offset: Number((page ?? 0) * rows),
                order_by: sortColumn ?? 'ban_asn_id',
                desc: sortOrder == 'desc',
                source_id: source_id,
                target_id: target_id
            });
        }
    });
    // const [newASNBans, setNewASNBans] = useState<ASNBanRecord[]>([]);

    // const onNewBanASN = useCallback(async () => {
    //     try {
    //         const ban = await NiceModal.show<ASNBanRecord>(ModalBanASN, {});
    //         setNewASNBans((prevState) => {
    //             return [ban, ...prevState];
    //         });
    //         sendFlash('success', `Created ASN ban successfully #${ban.ban_asn_id}`);
    //     } catch (e) {
    //         logErr(e);
    //     }
    // }, [sendFlash]);

    return (
        <Grid container>
            <Grid xs={12}>
                <ContainerWithHeaderAndButtons
                    title={'ASN Ban History'}
                    marginTop={0}
                    iconLeft={<GavelIcon />}
                    buttons={[
                        <Button
                            key={'btn-asn'}
                            variant={'contained'}
                            color={'success'}
                            startIcon={<AddIcon />}
                            sx={{ marginRight: 2 }}
                            // onClick={onNewBanASN}
                        >
                            Create
                        </Button>
                    ]}
                >
                    {/*<Formik*/}
                    {/*    initialValues={{*/}
                    {/*        as_num: Number(state.asNum),*/}
                    {/*        source_id: state.source,*/}
                    {/*        target_id: state.target,*/}
                    {/*        deleted: Boolean(state.deleted)*/}
                    {/*    }}*/}
                    {/*    onReset={onReset}*/}
                    {/*    onSubmit={onSubmit}*/}
                    {/*    validationSchema={validationSchema}*/}
                    {/*    validateOnChange={true}*/}
                    {/*>*/}
                    {/*    <Grid container spacing={3}>*/}
                    {/*        <Grid xs={12}>*/}
                    {/*            <Grid container spacing={2}>*/}
                    {/*                <Grid xs={4} sm={3} md={2}>*/}
                    {/*                    <ASNumberField />*/}
                    {/*                </Grid>*/}
                    {/*                <Grid xs={4} sm={3} md={2}>*/}
                    {/*                    <SourceIDField />*/}
                    {/*                </Grid>*/}
                    {/*                <Grid xs={4} sm={3} md={2}>*/}
                    {/*                    <TargetIDField />*/}
                    {/*                </Grid>*/}
                    {/*                <Grid xs={4} sm={3} md={2}>*/}
                    {/*                    <DeletedField />*/}
                    {/*                </Grid>*/}
                    {/*                <Grid xs={4} sm={3} md={2}>*/}
                    {/*                    <FilterButtons />*/}
                    {/*                </Grid>*/}
                    {/*            </Grid>*/}
                    {/*        </Grid>*/}
                    {/*        <Grid xs={12}>*/}
                    {/*<LazyTable<ASNBanRecord>*/}
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
                    {/*    onRowsPerPageChange={(*/}
                    {/*        event: ChangeEvent<*/}
                    {/*            HTMLInputElement | HTMLTextAreaElement*/}
                    {/*        >*/}
                    {/*    ) => {*/}
                    {/*        setState({*/}
                    {/*            rows: Number(event.target.value),*/}
                    {/*            page: 0*/}
                    {/*        });*/}
                    {/*    }}*/}
                    {/*    columns={[*/}
                    {/*        {*/}
                    {/*            label: '#',*/}
                    {/*            tooltip: 'Ban ID',*/}
                    {/*            sortKey: 'ban_asn_id',*/}
                    {/*            sortable: true,*/}
                    {/*            align: 'left',*/}
                    {/*            renderer: (obj) => (*/}
                    {/*                <Typography variant={'body1'}>*/}
                    {/*                    #{obj.ban_asn_id.toString()}*/}
                    {/*                </Typography>*/}
                    {/*            )*/}
                    {/*        },*/}
                    {/*        {*/}
                    {/*            label: 'A',*/}
                    {/*            tooltip: 'Ban Author Name',*/}
                    {/*            sortKey: 'source_personaname',*/}
                    {/*            sortable: true,*/}
                    {/*            align: 'left',*/}
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
                    {/*            label: 'Name',*/}
                    {/*            tooltip: 'Persona Name',*/}
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
                    {/*            label: 'ASN',*/}
                    {/*            tooltip: 'Autonomous System Numbers',*/}
                    {/*            sortKey: 'as_num',*/}
                    {/*            sortable: true,*/}
                    {/*            align: 'left',*/}
                    {/*            renderer: (row) => (*/}
                    {/*                <Typography variant={'body1'}>*/}
                    {/*                    {row.as_num}*/}
                    {/*                </Typography>*/}
                    {/*            )*/}
                    {/*        },*/}
                    {/*        {*/}
                    {/*            label: 'Reason',*/}
                    {/*            tooltip: 'Reason',*/}
                    {/*            sortKey: 'reason',*/}
                    {/*            sortable: true,*/}
                    {/*            align: 'left',*/}
                    {/*            renderer: (row) => (*/}
                    {/*                <Tooltip*/}
                    {/*                    title={*/}
                    {/*                        row.reason == BanReason.Custom*/}
                    {/*                            ? row.reason_text*/}
                    {/*                            : BanReason[row.reason]*/}
                    {/*                    }*/}
                    {/*                >*/}
                    {/*                    <Typography variant={'body1'}>*/}
                    {/*                        {BanReason[row.reason]}*/}
                    {/*                    </Typography>*/}
                    {/*                </Tooltip>*/}
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
                    {/*                return (*/}
                    {/*                    <Typography variant={'body1'}>*/}
                    {/*                        {renderDate(obj.created_on)}*/}
                    {/*                    </Typography>*/}
                    {/*                );*/}
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
                    {/*                return (*/}
                    {/*                    <Typography variant={'body1'}>*/}
                    {/*                        {renderDate(obj.valid_until)}*/}
                    {/*                    </Typography>*/}
                    {/*                );*/}
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
                    {/*                const dur = intervalToDuration({*/}
                    {/*                    start: row.created_on,*/}
                    {/*                    end: row.valid_until*/}
                    {/*                });*/}
                    {/*                const durationText =*/}
                    {/*                    dur.years && dur.years > 5*/}
                    {/*                        ? 'Permanent'*/}
                    {/*                        : formatDuration(dur);*/}
                    {/*                return (*/}
                    {/*                    <Typography*/}
                    {/*                        variant={'body1'}*/}
                    {/*                        overflow={'hidden'}*/}
                    {/*                    >*/}
                    {/*                        {durationText}*/}
                    {/*                    </Typography>*/}
                    {/*                );*/}
                    {/*            }*/}
                    {/*        },*/}
                    {/*        {*/}
                    {/*            label: 'A',*/}
                    {/*            tooltip:*/}
                    {/*                'Is this ban active (not deleted/inactive/unbanned)',*/}
                    {/*            align: 'center',*/}
                    {/*            width: '50px',*/}
                    {/*            sortKey: 'deleted',*/}
                    {/*            renderer: (row) => (*/}
                    {/*                <TableCellBool enabled={!row.deleted} />*/}
                    {/*            )*/}
                    {/*        },*/}
                    {/*        {*/}
                    {/*            label: 'Act.',*/}
                    {/*            tooltip: 'Actions',*/}
                    {/*            sortKey: 'reason',*/}
                    {/*            sortable: false,*/}
                    {/*            align: 'left',*/}
                    {/*            renderer: (row) => (*/}
                    {/*                <ButtonGroup fullWidth>*/}
                    {/*                    <IconButton*/}
                    {/*                        color={'warning'}*/}
                    {/*                        onClick={async () =>*/}
                    {/*                            await onEditASN(row)*/}
                    {/*                        }*/}
                    {/*                    >*/}
                    {/*                        <Tooltip title={'Edit ASN Ban'}>*/}
                    {/*                            <EditIcon />*/}
                    {/*                        </Tooltip>*/}
                    {/*                    </IconButton>*/}
                    {/*                    <IconButton*/}
                    {/*                        color={'success'}*/}
                    {/*                        onClick={async () =>*/}
                    {/*                            await onUnbanASN(row.as_num)*/}
                    {/*                        }*/}
                    {/*                    >*/}
                    {/*                        <Tooltip title={'Remove CIDR Ban'}>*/}
                    {/*                            <UndoIcon />*/}
                    {/*                        </Tooltip>*/}
                    {/*                    </IconButton>*/}
                    {/*                </ButtonGroup>*/}
                    {/*            )*/}
                    {/*        }*/}
                    {/*    ]}*/}
                    {/*/>*/}
                    <BanASMTable bans={bans ?? { data: [], count: 0 }} isLoading={isLoading} />
                </ContainerWithHeaderAndButtons>
            </Grid>
        </Grid>
    );
}

const columnHelper = createColumnHelper<ASNBanRecord>();

const BanASMTable = ({ bans, isLoading }: { bans: LazyResult<ASNBanRecord>; isLoading: boolean }) => {
    const columns = [
        columnHelper.accessor('ban_asn_id', {
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
        columnHelper.accessor('as_num', {
            header: () => <HeadingCell name={'ASN'} />,
            cell: (info) => <Typography>{info.getValue()}</Typography>
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
