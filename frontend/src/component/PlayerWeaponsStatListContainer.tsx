import { useState } from 'react';
import Link from '@mui/material/Link';
import { useQuery } from '@tanstack/react-query';
import { createColumnHelper, getCoreRowModel, getPaginationRowModel, useReactTable } from '@tanstack/react-table';
import { apiGetPlayerWeaponsOverall, WeaponsOverallResult } from '../api';
import { RowsPerPage } from '../util/table.ts';
import { defaultFloatFmt, defaultFloatFmtPct, humanCount } from '../util/text.tsx';
import { DataTable } from './DataTable.tsx';
import FmtWhenGt from './FmtWhenGT.tsx';
import { PaginatorLocal } from './PaginatorLocal.tsx';
import RouterLink from './RouterLink.tsx';

const columnHelper = createColumnHelper<WeaponsOverallResult>();

export const PlayerWeaponsStatListContainer = ({ steamId }: { steamId: string }) => {
    const [pagination, setPagination] = useState({
        pageIndex: 0, //initial page index
        pageSize: RowsPerPage.TwentyFive //default page size
    });

    const { data: stats, isLoading } = useQuery({
        queryKey: ['playerStats', { steamId }],
        queryFn: async () => {
            return await apiGetPlayerWeaponsOverall(steamId);
        }
    });

    const columns = [
        columnHelper.accessor('name', {
            header: 'Weapon',
            size: 350,
            cell: (info) => (
                <Link component={RouterLink} to={`/stats/weapon/${info.row.original.weapon_id}`}>
                    {String(info.getValue())}
                </Link>
            )
        }),
        columnHelper.accessor('kills', {
            header: 'Kills',
            cell: (info) => FmtWhenGt(info.getValue(), humanCount)
        }),
        columnHelper.accessor('kills_pct', {
            header: 'Kills%',
            cell: (info) => FmtWhenGt(info.getValue(), defaultFloatFmtPct)
        }),

        columnHelper.accessor('shots', {
            header: 'Shots',
            cell: (info) => FmtWhenGt(info.getValue(), humanCount)
        }),
        columnHelper.accessor('shots_pct', {
            header: 'Shot%',
            cell: (info) => FmtWhenGt(info.getValue(), defaultFloatFmtPct)
        }),
        columnHelper.accessor('hits', {
            header: 'Hits',
            cell: (info) => FmtWhenGt(info.getValue(), defaultFloatFmt)
        }),
        columnHelper.accessor('hits_pct', {
            header: 'Hits%',
            cell: (info) => FmtWhenGt(info.getValue(), defaultFloatFmtPct)
        }),
        columnHelper.accessor('accuracy', {
            header: 'Acc%',
            cell: (info) => FmtWhenGt(info.getValue(), defaultFloatFmtPct)
        }),
        columnHelper.accessor('airshots', {
            header: 'As',
            cell: (info) => FmtWhenGt(info.getValue(), humanCount)
        }),
        columnHelper.accessor('airshots_pct', {
            header: 'As%',
            cell: (info) => FmtWhenGt(info.getValue(), defaultFloatFmtPct)
        }),
        columnHelper.accessor('backstabs', {
            header: 'Bs',
            cell: (info) => FmtWhenGt(info.getValue(), humanCount)
        }),
        columnHelper.accessor('backstabs_pct', {
            header: 'Bs%',
            cell: (info) => FmtWhenGt(info.getValue(), defaultFloatFmtPct)
        }),
        columnHelper.accessor('headshots', {
            header: 'Hs',
            cell: (info) => FmtWhenGt(info.getValue(), humanCount)
        }),
        columnHelper.accessor('headshots_pct', {
            header: 'Hs%',
            cell: (info) => FmtWhenGt(info.getValue(), defaultFloatFmtPct)
        }),
        columnHelper.accessor('damage', {
            header: 'Dmg',
            cell: (info) => FmtWhenGt(info.getValue(), humanCount)
        }),
        columnHelper.accessor('damage_pct', {
            header: 'Dmg%',
            cell: (info) => FmtWhenGt(info.getValue(), defaultFloatFmtPct)
        })
    ];

    const table = useReactTable({
        data: stats?.data ?? [],
        columns: columns,
        getCoreRowModel: getCoreRowModel(),
        getPaginationRowModel: getPaginationRowModel(),
        onPaginationChange: setPagination, //update the pagination state when internal APIs mutate the pagination state
        state: {
            pagination
        }
    });

    return (
        <>
            <DataTable table={table} isLoading={isLoading} />
            <PaginatorLocal
                onRowsChange={(rows) => {
                    setPagination((prev) => {
                        return { ...prev, pageSize: rows };
                    });
                }}
                onPageChange={(page) => {
                    setPagination((prev) => {
                        return { ...prev, pageIndex: page };
                    });
                }}
                count={stats?.count ?? 0}
                rows={pagination.pageSize}
                page={pagination.pageIndex}
            />
        </>
    );
};
