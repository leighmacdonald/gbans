import AddIcon from '@mui/icons-material/Add';
import GavelIcon from '@mui/icons-material/Gavel';
import Button from '@mui/material/Button';
import TableCell from '@mui/material/TableCell';
import Typography from '@mui/material/Typography';
import Grid from '@mui/material/Unstable_Grid2';
import { useQuery } from '@tanstack/react-query';
import { createFileRoute } from '@tanstack/react-router';
import { createColumnHelper, getCoreRowModel, useReactTable } from '@tanstack/react-table';
import { z } from 'zod';
import { apiGetBansGroups, BanReasons, GroupBanRecord } from '../api';
import { ContainerWithHeaderAndButtons } from '../component/ContainerWithHeaderAndButtons.tsx';
import { DataTable, HeadingCell } from '../component/DataTable.tsx';
import { Paginator } from '../component/Paginator.tsx';
import { PersonCell } from '../component/PersonCell.tsx';
import { TableCellRelativeDateField } from '../component/table/TableCellRelativeDateField.tsx';
import { commonTableSearchSchema, isPermanentBan, LazyResult } from '../util/table.ts';
import { renderDate } from '../util/text.tsx';

const banGroupSearchSchema = z.object({
    ...commonTableSearchSchema,
    sortColumn: z.enum(['ban_group_id', 'source_id', 'target_id', 'deleted', 'reason', 'valid_until']).catch('ban_group_id'),
    source_id: z.string().catch(''),
    target_id: z.string().catch(''),
    group_id: z.number().catch(0),
    deleted: z.boolean().catch(false)
});

export const Route = createFileRoute('/_mod/admin/ban/group')({
    component: AdminBanGroup,
    validateSearch: (search) => banGroupSearchSchema.parse(search)
});

function AdminBanGroup() {
    const { page, rows, sortOrder, sortColumn, target_id, source_id } = Route.useSearch();
    const { data: bans, isLoading } = useQuery({
        queryKey: ['steamBans', { page, rows, sortOrder, sortColumn, target_id, source_id }],
        queryFn: async () => {
            return await apiGetBansGroups({
                limit: Number(rows),
                offset: Number((page ?? 0) * rows),
                order_by: sortColumn ?? 'ban_group_id',
                desc: sortOrder == 'desc',
                source_id: source_id,
                target_id: target_id
            });
        }
    });
    // const [newGroupBans, setNewGroupBans] = useState<GroupBanRecord[]>([]);
    // const { sendFlash } = useUserFlashCtx();

    // const onNewBanGroup = useCallback(async () => {
    //     try {
    //         const ban = await NiceModal.show<GroupBanRecord>(ModalBanGroup, {});
    //         setNewGroupBans((prevState) => {
    //             return [ban, ...prevState];
    //         });
    //         sendFlash('success', `Created steam group ban successfully #${ban.ban_group_id}`);
    //     } catch (e) {
    //         logErr(e);
    //     }
    // }, [sendFlash]);

    return (
        <Grid container>
            <Grid xs={12}>
                <ContainerWithHeaderAndButtons
                    title={'Steam Group Ban History'}
                    marginTop={0}
                    iconLeft={<GavelIcon />}
                    buttons={[
                        <Button
                            key={`ban-group`}
                            variant={'contained'}
                            color={'success'}
                            startIcon={<AddIcon />}
                            sx={{ marginRight: 2 }}
                            // onClick={onNewBanGroup}
                        >
                            Create
                        </Button>
                    ]}
                >
                    {/*<BanGroupTable newBans={newGroupBans} />*/}
                    <BanGroupTable bans={bans ?? { data: [], count: 0 }} isLoading={isLoading} />
                    <Paginator page={page} rows={rows} data={bans} />
                </ContainerWithHeaderAndButtons>
            </Grid>
        </Grid>
    );
}

const columnHelper = createColumnHelper<GroupBanRecord>();

const BanGroupTable = ({ bans, isLoading }: { bans: LazyResult<GroupBanRecord>; isLoading: boolean }) => {
    const columns = [
        columnHelper.accessor('ban_group_id', {
            header: () => <HeadingCell name={'Ban ID'} />,
            cell: (info) => <TableCell>{`#${info.getValue()}`}</TableCell>
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
        columnHelper.accessor('group_id', {
            header: () => <HeadingCell name={'Group'} />,
            cell: (info) => <Typography>{info.getValue()}</Typography>
        }),
        columnHelper.accessor('note', {
            header: () => <HeadingCell name={'Note'} />,
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
