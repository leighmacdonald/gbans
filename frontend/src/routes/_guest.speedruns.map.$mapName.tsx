import { useState } from 'react';
import EmojiEventsIcon from '@mui/icons-material/EmojiEvents';
import Typography from '@mui/material/Typography';
import { useQuery } from '@tanstack/react-query';
import { createFileRoute } from '@tanstack/react-router';
import {
    createColumnHelper,
    getCoreRowModel,
    getPaginationRowModel,
    TableOptions,
    useReactTable
} from '@tanstack/react-table';
import { apiGetServers, getSpeedrunsTopMap } from '../api';
import { ContainerWithHeader } from '../component/ContainerWithHeader.tsx';
import { TextLink } from '../component/TextLink.tsx';
import { Title } from '../component/Title';
import { PaginatorLocal } from '../component/forum/PaginatorLocal.tsx';
import { DataTable } from '../component/table/DataTable.tsx';
import { TableCellSmall } from '../component/table/TableCellSmall.tsx';
import { TableCellString } from '../component/table/TableCellString.tsx';
import { ServerSimple } from '../schema/server.ts';
import { SpeedrunMapOverview } from '../schema/speedrun.ts';
import { ensureFeatureEnabled } from '../util/features.ts';
import { RowsPerPage } from '../util/table.ts';
import { renderDateTime, durationString } from '../util/time.ts';

export const Route = createFileRoute('/_guest/speedruns/map/$mapName')({
    component: SpeedrunsMap,
    beforeLoad: () => {
        ensureFeatureEnabled('speedruns_enabled');
    }
});

function SpeedrunsMap() {
    const { mapName } = Route.useParams();
    const title = `Speedruns: ${mapName}`;

    const { data: speedruns, isLoading } = useQuery({
        queryKey: ['speedruns_map', mapName],
        queryFn: () => {
            return getSpeedrunsTopMap(mapName);
        }
    });

    const { data: servers, isLoading: isLoadingServers } = useQuery({
        queryKey: ['serversSimple'],
        queryFn: apiGetServers
    });

    return (
        <>
            <Title>{title}</Title>
            <ContainerWithHeader title={title} iconLeft={<EmojiEventsIcon />}>
                <SpeedrunTable
                    speedruns={speedruns ?? []}
                    servers={servers ?? []}
                    isLoading={isLoading || isLoadingServers}
                ></SpeedrunTable>
            </ContainerWithHeader>
        </>
    );
}

const columnHelper = createColumnHelper<SpeedrunMapOverview>();

const SpeedrunTable = ({
    speedruns,
    servers,
    isLoading
}: {
    speedruns: SpeedrunMapOverview[];
    servers: ServerSimple[];
    isLoading: boolean;
}) => {
    const [pagination, setPagination] = useState({
        pageIndex: 0,
        pageSize: RowsPerPage.TwentyFive
    });
    const columns = [
        columnHelper.accessor('rank', {
            header: 'Rank',
            size: 5,
            cell: (info) => {
                const value = info.getValue();
                const weight = value <= 3 ? 700 : 500;
                return (
                    <TextLink
                        fontWeight={weight}
                        textAlign={'right'}
                        paddingRight={2}
                        to={'/speedruns/id/$speedrunId'}
                        params={{ speedrunId: String(info.row.original.speedrun_id) }}
                    >
                        {value}
                    </TextLink>
                );
            }
        }),
        columnHelper.accessor('initial_rank', {
            header: 'High',
            size: 10,
            cell: (info) => (
                <TableCellSmall>
                    <Typography align={'center'}>{info.getValue()}</Typography>
                </TableCellSmall>
            )
        }),
        columnHelper.accessor('duration', {
            header: 'Time',
            size: 100,
            cell: (info) => (
                <TableCellSmall>
                    <Typography align={'center'}>{durationString(info.getValue() / 1000)}</Typography>
                </TableCellSmall>
            )
        }),
        columnHelper.accessor('player_count', {
            header: 'Max Players',
            size: 100,
            cell: (info) => {
                return <TableCellString>{info.getValue()}</TableCellString>;
            }
        }),
        columnHelper.accessor('bot_count', {
            header: 'Max Bots',
            size: 100,
            cell: (info) => {
                return <TableCellString>{info.getValue()}</TableCellString>;
            }
        }),
        columnHelper.accessor('total_players', {
            header: 'Total Players',
            size: 100,
            cell: (info) => {
                return <TableCellString>{info.getValue()}</TableCellString>;
            }
        }),
        columnHelper.accessor('server_id', {
            header: 'Server',
            size: 100,
            cell: (info) => {
                const srv = (servers ?? []).find((s) => (s.server_id = info.getValue()));
                return <TableCellString>{srv?.server_name}</TableCellString>;
            }
        }),
        columnHelper.accessor('created_on', {
            header: 'Submitted',
            size: 100,
            cell: (info) => {
                return <TableCellString>{renderDateTime(info.getValue())}</TableCellString>;
            }
        })
    ];

    const opts: TableOptions<SpeedrunMapOverview> = {
        data: speedruns,
        columns: columns,
        getCoreRowModel: getCoreRowModel(),
        manualPagination: false,
        autoResetPageIndex: true,
        onPaginationChange: setPagination,
        getPaginationRowModel: getPaginationRowModel(),
        state: { pagination }
    };

    const table = useReactTable(opts);

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
                count={speedruns.length}
                rows={pagination.pageSize}
                page={pagination.pageIndex}
            />
        </>
    );
};
