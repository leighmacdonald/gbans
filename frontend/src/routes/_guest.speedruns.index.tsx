import { useMemo, useState } from 'react';
import ElectricBoltIcon from '@mui/icons-material/ElectricBolt';
import EmojiEventsIcon from '@mui/icons-material/EmojiEvents';
import HistoryIcon from '@mui/icons-material/History';
import PageviewIcon from '@mui/icons-material/Pageview';
import ButtonGroup from '@mui/material/ButtonGroup';
import Grid from '@mui/material/Grid';
import TableCell from '@mui/material/TableCell';
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
import {
    apiGetServers,
    getSpeedrunsRecent,
    getSpeedrunsTopOverall,
    ServerSimple,
    SpeedrunMapOverview,
    SpeedrunResult
} from '../api';
import { ButtonLink } from '../component/ButtonLink.tsx';
import { ContainerWithHeader } from '../component/ContainerWithHeader.tsx';
import { ContainerWithHeaderAndButtons } from '../component/ContainerWithHeaderAndButtons.tsx';
import { TextLink } from '../component/TextLink.tsx';
import { Title } from '../component/Title';
import { DataTable } from '../component/table/DataTable.tsx';
import { TableCellSmall } from '../component/table/TableCellSmall.tsx';
import { TableCellString } from '../component/table/TableCellString.tsx';
import { RowsPerPage } from '../util/table.ts';
import { renderDateTime, durationString } from '../util/time.ts';

export const Route = createFileRoute('/_guest/speedruns/')({
    component: SpeedrunsOverall
});

const columnHelper = createColumnHelper<SpeedrunResult>();

function SpeedrunsOverall() {
    const recentCount = 10;
    const { data: speedruns, isLoading } = useQuery({
        queryKey: ['speedruns_overall'],
        queryFn: () => {
            return getSpeedrunsTopOverall(10);
        }
    });

    const { data: recent, isLoading: isLoadingRecent } = useQuery({
        queryKey: ['speedruns_recent', recentCount],
        queryFn: async () => {
            return await getSpeedrunsRecent(recentCount);
        }
    });

    const { data: servers, isLoading: isLoadingServers } = useQuery({
        queryKey: ['serversSimple'],
        queryFn: apiGetServers
    });

    const keys = useMemo(() => {
        if (!speedruns) {
            return [];
        }
        return Object.keys(speedruns).sort();
    }, [speedruns]);

    return (
        <>
            <Title>Speedrun Overall Results</Title>
            <Grid container spacing={2}>
                <Grid size={{ xs: 12, md: 4 }}>
                    <ContainerWithHeader title={'Speedruns'} iconLeft={<ElectricBoltIcon />}>
                        <Typography>
                            These are the overall results for the speedruns. Speedruns are automatically created upon
                            match completion. For a player to count in the overall participants, they must have played a
                            minimum of 25% of the total play time of the map.
                        </Typography>
                    </ContainerWithHeader>
                </Grid>

                <Grid size={{ xs: 12, md: 8 }}>
                    <ContainerWithHeader title={'Most Recent Changes'} iconLeft={<HistoryIcon />}>
                        <SpeedrunRecentTable
                            speedruns={recent ?? []}
                            isLoading={isLoadingRecent}
                            servers={servers ?? []}
                        />
                    </ContainerWithHeader>
                </Grid>

                {speedruns &&
                    keys.map((map_name) => {
                        return (
                            <Grid size={{ xs: 12, md: 6, lg: 4 }} key={`map-${map_name}`}>
                                <ContainerWithHeaderAndButtons
                                    title={map_name}
                                    iconLeft={<EmojiEventsIcon />}
                                    buttons={[
                                        <ButtonGroup key={'buttons'}>
                                            <ButtonLink
                                                variant={'contained'}
                                                color={'success'}
                                                endIcon={<PageviewIcon />}
                                                to={'/speedruns/map/$mapName'}
                                                params={{ mapName: map_name }}
                                            >
                                                More
                                            </ButtonLink>
                                        </ButtonGroup>
                                    ]}
                                >
                                    <SpeedrunTopTable
                                        speedruns={speedruns[map_name]}
                                        servers={servers}
                                        isLoading={isLoading || isLoadingServers}
                                    ></SpeedrunTopTable>
                                </ContainerWithHeaderAndButtons>
                            </Grid>
                        );
                    })}
            </Grid>
        </>
    );
}

const SpeedrunRecentTable = ({
    speedruns,
    servers,
    isLoading
}: {
    speedruns: SpeedrunMapOverview[];
    servers?: ServerSimple[];
    isLoading: boolean;
}) => {
    const [pagination, setPagination] = useState({
        pageIndex: 0,
        pageSize: RowsPerPage.Ten
    });
    const ch = createColumnHelper<SpeedrunMapOverview>();
    const columns = [
        ch.accessor('rank', {
            header: 'Rank',
            size: 10,
            cell: (info) => {
                const value = info.getValue();
                const weight = value <= 3 ? 700 : 500;
                return (
                    <TableCell>
                        <Typography fontWeight={weight}>{value}</Typography>
                    </TableCell>
                );
            }
        }),
        ch.accessor('speedrun_id', {
            header: 'ID',
            size: 10,
            cell: (info) => {
                return (
                    <TableCell>
                        <TextLink
                            fontWeight={700}
                            to={'/speedruns/id/$speedrunId'}
                            params={{ speedrunId: String(info.row.original.speedrun_id) }}
                        >
                            {info.getValue()}
                        </TextLink>
                    </TableCell>
                );
            }
        }),
        ch.accessor('map_detail', {
            header: 'Map',
            size: 60,
            cell: (info) => (
                <TableCellSmall>
                    <Typography align={'center'}>{info.getValue().map_name}</Typography>
                </TableCellSmall>
            )
        }),
        ch.accessor('duration', {
            header: 'Time',
            size: 60,
            cell: (info) => (
                <TableCellSmall>
                    <Typography align={'center'}>{durationString(info.getValue())}</Typography>
                </TableCellSmall>
            )
        }),
        ch.accessor('total_players', {
            header: 'Players',
            size: 30,
            cell: (info) => {
                return <TableCellString>{info.row.original.total_players}</TableCellString>;
            }
        }),
        ch.accessor('server_id', {
            header: 'Server',
            size: 30,
            cell: (info) => {
                const srv = (servers ?? []).find((s) => (s.server_id = info.getValue()));
                return <TableCellString>{srv?.server_name}</TableCellString>;
            }
        }),
        ch.accessor('created_on', {
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

    return <DataTable table={table} isLoading={isLoading} />;
};

const SpeedrunTopTable = ({
    speedruns,
    servers,
    isLoading
}: {
    speedruns: SpeedrunResult[];
    servers?: ServerSimple[];
    isLoading: boolean;
}) => {
    const [pagination, setPagination] = useState({
        pageIndex: 0,
        pageSize: RowsPerPage.TwentyFive
    });
    const columns = [
        columnHelper.accessor('rank', {
            header: 'Rank',
            size: 10,
            cell: (info) => {
                const value = info.getValue();
                const weight = value <= 3 ? 700 : 500;
                return (
                    <TableCell>
                        <TextLink
                            fontWeight={weight}
                            to={'/speedruns/id/$speedrunId'}
                            params={{ speedrunId: String(info.row.original.speedrun_id) }}
                        >
                            {value}
                        </TextLink>
                    </TableCell>
                );
            }
        }),
        columnHelper.accessor('duration', {
            header: 'Time',
            size: 60,
            cell: (info) => (
                <TableCellSmall>
                    <Typography align={'center'}>{durationString(info.getValue())}</Typography>
                </TableCellSmall>
            )
        }),
        columnHelper.accessor('players', {
            header: 'Players',
            size: 30,
            cell: (info) => {
                return <TableCellString>{info.getValue().length}</TableCellString>;
            }
        }),
        columnHelper.accessor('server_id', {
            header: 'Srv',
            size: 30,
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

    const opts: TableOptions<SpeedrunResult> = {
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

    return <DataTable table={table} isLoading={isLoading} />;
};
