import AddIcon from '@mui/icons-material/Add';
import GavelIcon from '@mui/icons-material/Gavel';
import Box from '@mui/material/Box';
import Button from '@mui/material/Button';
import Typography from '@mui/material/Typography';
import Grid from '@mui/material/Unstable_Grid2';
import { useQuery } from '@tanstack/react-query';
import { createFileRoute } from '@tanstack/react-router';
import { createColumnHelper, getCoreRowModel, useReactTable } from '@tanstack/react-table';
import { z } from 'zod';
import { apiGetBansCIDR, BanReason, BanReasons, CIDRBanRecord } from '../api';
import { ContainerWithHeaderAndButtons } from '../component/ContainerWithHeaderAndButtons.tsx';
import { DataTable, HeadingCell } from '../component/DataTable.tsx';
import { Paginator } from '../component/Paginator.tsx';
import { PersonCell } from '../component/PersonCell.tsx';
import { TableCellBool } from '../component/table/TableCellBool.tsx';
import { TableCellRelativeDateField } from '../component/table/TableCellRelativeDateField.tsx';
import { commonTableSearchSchema, isPermanentBan, LazyResult } from '../util/table.ts';
import { renderDate } from '../util/text.tsx';

const banCIDRSearchSchema = z.object({
    ...commonTableSearchSchema,
    sortColumn: z.enum(['net_id', 'source_id', 'target_id', 'deleted', 'reason', 'created_on', 'valid_until']).catch('net_id'),
    source_id: z.string().catch(''),
    target_id: z.string().catch(''),
    cidr: z.string().optional().catch(''),
    reason: z.nativeEnum(BanReason).optional()
});

export const Route = createFileRoute('/_mod/admin/ban/cidr')({
    component: AdminBanCIDR,
    validateSearch: (search) => banCIDRSearchSchema.parse(search)
});

function AdminBanCIDR() {
    const { page, rows, sortOrder, sortColumn, target_id, source_id } = Route.useSearch();
    const { data: bans, isLoading } = useQuery({
        queryKey: ['steamBans', { page, rows, sortOrder, sortColumn, target_id, source_id }],
        queryFn: async () => {
            return await apiGetBansCIDR({
                limit: Number(rows),
                offset: Number((page ?? 0) * rows),
                order_by: sortColumn,
                desc: sortOrder == 'desc',
                source_id: source_id,
                target_id: target_id
            });
        }
    });
    // const [newCIDRBans, setNewCIDRBans] = useState<CIDRBanRecord[]>([]);
    // const { sendFlash } = useUserFlashCtx();

    // const onNewBanCIDR = useCallback(async () => {
    //     try {
    //         const ban = await NiceModal.show<CIDRBanRecord>(ModalBanCIDR, {});
    //         setNewCIDRBans((prevState) => {
    //             return [ban, ...prevState];
    //         });
    //         sendFlash('success', `Created CIDR ban successfully #${ban.net_id}`);
    //     } catch (e) {
    //         logErr(e);
    //     }
    // }, [sendFlash]);

    return (
        <Grid container>
            <Grid xs={12} marginBottom={2}>
                <Box></Box>
            </Grid>
            <Grid xs={12}>
                <ContainerWithHeaderAndButtons
                    title={'CIDR Ban History'}
                    marginTop={0}
                    iconLeft={<GavelIcon />}
                    buttons={[
                        <Button
                            key={'btn-cidr'}
                            variant={'contained'}
                            color={'success'}
                            startIcon={<AddIcon />}
                            sx={{ marginRight: 2 }}
                            // onClick={onNewBanCIDR}
                        >
                            Create
                        </Button>
                    ]}
                >
                    <BanCIDRTable bans={bans ?? { data: [], count: 0 }} isLoading={isLoading} />
                    <Paginator data={bans} page={page} rows={rows} />
                </ContainerWithHeaderAndButtons>
            </Grid>
        </Grid>
    );
}

const columnHelper = createColumnHelper<CIDRBanRecord>();

const BanCIDRTable = ({ bans, isLoading }: { bans: LazyResult<CIDRBanRecord>; isLoading: boolean }) => {
    const columns = [
        columnHelper.accessor('net_id', {
            header: () => <HeadingCell name={'Ban ID'} />,
            cell: (info) => <Typography>{`#${info.getValue()}`}</Typography>
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
        columnHelper.accessor('cidr', {
            header: () => <HeadingCell name={'CIDR (hosts)'} />,
            cell: (info) => <Typography>{`${info.getValue()}`}</Typography>
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
        columnHelper.accessor('deleted', {
            header: () => <HeadingCell name={'D'} />,
            cell: (info) => <TableCellBool enabled={info.getValue()} />
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
