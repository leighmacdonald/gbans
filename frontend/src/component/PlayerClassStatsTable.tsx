import Typography from '@mui/material/Typography';
import { useQuery } from '@tanstack/react-query';
import { createColumnHelper, getCoreRowModel, useReactTable } from '@tanstack/react-table';
import { formatDistance } from 'date-fns';
import { apiGetPlayerClassOverallStats, PlayerClassOverallResult } from '../api';
import { defaultFloatFmt, humanCount } from '../util/text.tsx';
import { DataTable } from './DataTable.tsx';
import FmtWhenGt from './FmtWhenGT.tsx';
import { PlayerClassImg } from './PlayerClassImg';
import { TableCellSmall } from './TableCellSmall.tsx';
import { TableHeadingCell } from './TableHeadingCell.tsx';

interface PlayerClassStatsContainerProps {
    steam_id: string;
}

const columnHelper = createColumnHelper<PlayerClassOverallResult>();

export const PlayerClassStatsTable = ({ steam_id }: PlayerClassStatsContainerProps) => {
    const { data: stats, isLoading } = useQuery({
        queryKey: ['playerStats', { steam_id }],
        queryFn: async () => {
            return await apiGetPlayerClassOverallStats(steam_id);
        }
    });

    const columns = [
        columnHelper.accessor('player_class_id', {
            header: () => <TableHeadingCell name={'Class'} />,
            cell: (info) => (
                <TableCellSmall>
                    <PlayerClassImg cls={info.getValue()} />
                </TableCellSmall>
            )
        }),
        columnHelper.accessor('playtime', {
            header: () => <TableHeadingCell name={'Playtime'} />,
            cell: (info) => (
                <TableCellSmall>
                    {formatDistance(0, info.getValue() * 1000, {
                        includeSeconds: true
                    })}
                </TableCellSmall>
            )
        }),
        columnHelper.accessor('ka', {
            header: () => <TableHeadingCell name={'KA'} />,
            cell: (info) => (
                <TableCellSmall>
                    <Typography>{FmtWhenGt(info.getValue(), humanCount)}</Typography>
                </TableCellSmall>
            )
        }),
        columnHelper.accessor('kills', {
            header: () => <TableHeadingCell name={'K'} />,
            cell: (info) => (
                <TableCellSmall>
                    <Typography>{FmtWhenGt(info.getValue(), humanCount)}</Typography>
                </TableCellSmall>
            )
        }),
        columnHelper.accessor('assists', {
            header: () => <TableHeadingCell name={'A'} />,
            cell: (info) => (
                <TableCellSmall>
                    <Typography>{FmtWhenGt(info.getValue(), humanCount)}</Typography>
                </TableCellSmall>
            )
        }),
        columnHelper.accessor('deaths', {
            header: () => <TableHeadingCell name={'D'} />,
            cell: (info) => (
                <TableCellSmall>
                    <Typography>{FmtWhenGt(info.getValue(), humanCount)}</Typography>
                </TableCellSmall>
            )
        }),
        columnHelper.accessor('kad', {
            header: () => <TableHeadingCell name={'KAD'} />,
            cell: (info) => (
                <TableCellSmall>
                    <Typography>{FmtWhenGt(info.getValue(), defaultFloatFmt)}</Typography>
                </TableCellSmall>
            )
        }),
        columnHelper.accessor('damage', {
            header: () => <TableHeadingCell name={'Dmg'} />,
            cell: (info) => (
                <TableCellSmall>
                    <Typography>{FmtWhenGt(info.getValue(), humanCount)}</Typography>
                </TableCellSmall>
            )
        }),
        columnHelper.accessor('dpm', {
            header: () => <TableHeadingCell name={'Dpm'} />,
            cell: (info) => (
                <TableCellSmall>
                    <Typography>{FmtWhenGt(info.getValue(), defaultFloatFmt)}</Typography>
                </TableCellSmall>
            )
        }),
        columnHelper.accessor('damage_taken', {
            header: () => <TableHeadingCell name={'DT'} />,
            cell: (info) => (
                <TableCellSmall>
                    <Typography>{FmtWhenGt(info.getValue(), humanCount)}</Typography>
                </TableCellSmall>
            )
        }),
        columnHelper.accessor('dominations', {
            header: () => <TableHeadingCell name={'DM'} />,
            cell: (info) => (
                <TableCellSmall>
                    <Typography>{FmtWhenGt(info.getValue(), humanCount)}</Typography>
                </TableCellSmall>
            )
        }),
        columnHelper.accessor('dominated', {
            header: () => <TableHeadingCell name={'DD'} />,
            cell: (info) => (
                <TableCellSmall>
                    <Typography>{FmtWhenGt(info.getValue(), humanCount)}</Typography>
                </TableCellSmall>
            )
        }),
        columnHelper.accessor('revenges', {
            header: () => <TableHeadingCell name={'Rv'} />,
            cell: (info) => (
                <TableCellSmall>
                    <Typography>{FmtWhenGt(info.getValue(), humanCount)}</Typography>
                </TableCellSmall>
            )
        }),
        columnHelper.accessor('captures', {
            header: () => <TableHeadingCell name={'CP'} />,
            cell: (info) => (
                <TableCellSmall>
                    <Typography>{FmtWhenGt(info.getValue(), humanCount)}</Typography>
                </TableCellSmall>
            )
        })
    ];

    const table = useReactTable({
        data: stats?.data ?? [],
        columns: columns,
        getCoreRowModel: getCoreRowModel()
    });

    return (
        <>
            <DataTable table={table} isLoading={isLoading} />
        </>
    );
};
