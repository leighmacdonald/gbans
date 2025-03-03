import { useQuery } from '@tanstack/react-query';
import { createColumnHelper, getCoreRowModel, useReactTable } from '@tanstack/react-table';
import { formatDistance } from 'date-fns';
import { apiGetPlayerClassOverallStats, PlayerClassOverallResult } from '../api';
import { defaultFloatFmt, humanCount } from '../util/text.tsx';
import { DataTable } from './DataTable.tsx';
import FmtWhenGt from './FmtWhenGT.tsx';
import { PlayerClassImg } from './PlayerClassImg';

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
            header: 'Class',
            cell: (info) => <PlayerClassImg cls={info.getValue()} />
        }),
        columnHelper.accessor('playtime', {
            header: 'Playtime',
            cell: (info) =>
                formatDistance(0, info.getValue() * 1000, {
                    includeSeconds: true
                })
        }),
        columnHelper.accessor('ka', {
            header: 'KA',
            cell: (info) => FmtWhenGt(info.getValue(), humanCount)
        }),
        columnHelper.accessor('kills', {
            header: 'K',
            cell: (info) => FmtWhenGt(info.getValue(), humanCount)
        }),
        columnHelper.accessor('assists', {
            header: 'A',
            cell: (info) => FmtWhenGt(info.getValue(), humanCount)
        }),
        columnHelper.accessor('deaths', {
            header: 'D',
            cell: (info) => FmtWhenGt(info.getValue(), humanCount)
        }),
        columnHelper.accessor('kad', {
            header: 'KAD',
            cell: (info) => FmtWhenGt(info.getValue(), defaultFloatFmt)
        }),
        columnHelper.accessor('damage', {
            header: 'Dmg',
            cell: (info) => FmtWhenGt(info.getValue(), humanCount)
        }),
        columnHelper.accessor('dpm', {
            header: 'Dpm',
            cell: (info) => FmtWhenGt(info.getValue(), defaultFloatFmt)
        }),
        columnHelper.accessor('damage_taken', {
            header: 'DT',
            cell: (info) => FmtWhenGt(info.getValue(), humanCount)
        }),
        columnHelper.accessor('dominations', {
            header: 'DM',
            cell: (info) => FmtWhenGt(info.getValue(), humanCount)
        }),
        columnHelper.accessor('dominated', {
            header: 'DD',
            cell: (info) => FmtWhenGt(info.getValue(), humanCount)
        }),
        columnHelper.accessor('revenges', {
            header: 'Rv',
            cell: (info) => FmtWhenGt(info.getValue(), humanCount)
        }),
        columnHelper.accessor('captures', {
            header: 'CP',
            cell: (info) => FmtWhenGt(info.getValue(), humanCount)
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
