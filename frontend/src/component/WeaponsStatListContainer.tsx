import { useState } from 'react';
import InsightsIcon from '@mui/icons-material/Insights';
import { useQuery } from '@tanstack/react-query';
import { createColumnHelper, getCoreRowModel, getPaginationRowModel, useReactTable } from '@tanstack/react-table';
import { apiGetWeaponsOverall } from '../api';
import { WeaponsOverallResult } from '../schema/stats.ts';
import { LazyResult, RowsPerPage } from '../util/table.ts';
import { defaultFloatFmt, defaultFloatFmtPct, humanCount } from '../util/text.tsx';
import { ContainerWithHeader } from './ContainerWithHeader';
import FmtWhenGt from './FmtWhenGT.tsx';
import { TextLink } from './TextLink.tsx';
import { PaginatorLocal } from './forum/PaginatorLocal.tsx';
import { DataTable } from './table/DataTable.tsx';

export const WeaponsStatListContainer = () => {
    const { data, isLoading } = useQuery({
        queryKey: ['statsWeaponsOverall'],
        queryFn: async () => apiGetWeaponsOverall()
    });

    return (
        <ContainerWithHeader title={'Overall Weapon Stats'} iconLeft={<InsightsIcon />}>
            <StatsWeaponsOverall stats={data ?? { data: [], count: 0 }} isLoading={isLoading} />
        </ContainerWithHeader>
    );
};

const columnHelper = createColumnHelper<WeaponsOverallResult>();

const StatsWeaponsOverall = ({ stats, isLoading }: { stats: LazyResult<WeaponsOverallResult>; isLoading: boolean }) => {
    const [pagination, setPagination] = useState({
        pageIndex: 0, //initial page index
        pageSize: RowsPerPage.TwentyFive //default page size
    });

    const columns = [
        columnHelper.accessor('weapon_id', {
            header: 'Weapon',
            size: 350,
            cell: (info) => (
                <TextLink to={'/stats/weapon/$weapon_id'} params={{ weapon_id: String(info.getValue()) }}>
                    {stats.data[info.row.index].name}
                </TextLink>
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
            header: 'Shots%',
            cell: (info) => FmtWhenGt(info.getValue(), defaultFloatFmt)
        }),
        columnHelper.accessor('hits', {
            header: 'Hits',
            cell: (info) => FmtWhenGt(info.getValue(), humanCount)
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
        data: stats.data,
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
                count={stats.count}
                rows={pagination.pageSize}
                page={pagination.pageIndex}
            />
        </>
    );
};
