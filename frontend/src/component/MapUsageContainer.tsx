import { SyntheticEvent, useMemo, useState } from 'react';
import MapIcon from '@mui/icons-material/Map';
import TabContext from '@mui/lab/TabContext';
import TabList from '@mui/lab/TabList';
import TabPanel from '@mui/lab/TabPanel';
import Box from '@mui/material/Box';
import Grid from '@mui/material/Grid';
import Tab from '@mui/material/Tab';
import Typography from '@mui/material/Typography';
import { PieChart } from '@mui/x-charts';
import { useQuery } from '@tanstack/react-query';
import { createColumnHelper, getCoreRowModel, getPaginationRowModel, useReactTable } from '@tanstack/react-table';
import { apiGetMapUsage } from '../api';
import { RowsPerPage } from '../util/table.ts';
import { ContainerWithHeader } from './ContainerWithHeader';
import { DataTable } from './DataTable.tsx';
import { LoadingSpinner } from './LoadingSpinner';
import { PaginatorLocal } from './PaginatorLocal.tsx';
import { TableCellSmall } from './TableCellSmall.tsx';

interface MapUseChartProps {
    details: SeriesData[];
}

const MapUseChart = ({ details }: MapUseChartProps) => {
    const merged = useMemo(() => {
        if (details.length < 20) {
            return details;
        }
        const small: SeriesData = { value: 0, label: 'other', id: 'other' };
        const large: SeriesData[] = [];

        for (let i = 0; i < details.length; i++) {
            if (details[i].value < 1) {
                small.value += details[i].value;
            } else {
                large.push(details[i]);
            }
        }
        return [small, ...large];
    }, [details]);

    return (
        <PieChart
            height={600}
            width={600}
            slotProps={{ legend: { hidden: true } }}
            series={[
                {
                    data: merged,
                    highlightScope: { faded: 'global', highlighted: 'item' },
                    faded: { innerRadius: 30, additionalRadius: -30 },
                    valueFormatter: (value) => {
                        return `${value.value.toFixed(2)}%`;
                    }
                }
            ]}
        />
    );
};

interface SeriesData {
    id: string;
    label: string;
    value: number;
}

interface BarChartWithTableProps {
    isLoading: boolean;
    data: SeriesData[];
}

const BarChartWithTable = ({ isLoading, data }: BarChartWithTableProps) => {
    return (
        <Grid container>
            <Grid size={{ xs: 12, md: 6 }}>
                <Box paddingLeft={10} display="flex" justifyContent="center" alignItems="center">
                    {isLoading ? <LoadingSpinner /> : <MapUseChart details={data} />}
                </Box>
            </Grid>
            <Grid size={{ xs: 12, md: 6 }}>
                <SeriesTable stats={data} isLoading={isLoading} />
            </Grid>
        </Grid>
    );
};

export const MapUsageContainer = () => {
    const [value, setValue] = useState('1');

    const handleChange = (_: SyntheticEvent, newValue: string) => {
        setValue(newValue);
    };

    const { data: stats, isLoading } = useQuery({
        queryKey: ['mapStats'],
        queryFn: async () => {
            const resp = await apiGetMapUsage();
            const maps = resp.map((value1): SeriesData => {
                return {
                    id: value1.map,
                    value: value1.percent,
                    label: value1.map.replace('workshop/', '').split('.')[0]
                };
            });

            const mapsRecords: Record<string, number> = {};

            for (let i = 0; i < resp.length; i++) {
                const key = resp[i].map.replace('workshop/', '').split('_')[0];
                if (!mapsRecords[key]) {
                    mapsRecords[key] = 0;
                }
                mapsRecords[key] += resp[i].percent;
            }
            const modes: SeriesData[] = [];

            for (const mapsKey in mapsRecords) {
                modes.push({
                    label: mapsKey,
                    id: mapsKey,
                    value: mapsRecords[mapsKey]
                });
            }

            return { maps, modes };
        }
    });

    return (
        <ContainerWithHeader title={'Map Playtime Distribution'} iconLeft={<MapIcon />}>
            <TabContext value={value}>
                <Box sx={{ borderBottom: 1, borderColor: 'divider' }}>
                    <TabList onChange={handleChange} aria-label="lab API tabs example">
                        <Tab label="By Map" value="1" />
                        <Tab label="By Mode" value="2" />
                    </TabList>
                </Box>
                <TabPanel value="1">
                    <BarChartWithTable isLoading={isLoading} data={stats?.maps ?? []} />
                </TabPanel>
                <TabPanel value="2">
                    <BarChartWithTable isLoading={isLoading} data={stats?.modes ?? []} />
                </TabPanel>
            </TabContext>
        </ContainerWithHeader>
    );
};

const SeriesTable = ({ stats, isLoading }: { stats: SeriesData[]; isLoading: boolean }) => {
    const columnHelper = createColumnHelper<SeriesData>();

    const [pagination, setPagination] = useState({
        pageIndex: 0, //initial page index
        pageSize: RowsPerPage.TwentyFive //default page size
    });

    const columns = [
        columnHelper.accessor('label', {
            header: 'Name',
            cell: (info) => (
                <TableCellSmall>
                    <Typography>{info.getValue()}</Typography>
                </TableCellSmall>
            )
        }),

        columnHelper.accessor('value', {
            header: 'Value',
            size: 30,
            cell: (info) => (
                <TableCellSmall>
                    <Typography>{info.getValue().toFixed(2)} %</Typography>
                </TableCellSmall>
            )
        })
    ];

    const table = useReactTable({
        data: stats,
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
                count={stats.length}
                rows={pagination.pageSize}
                page={pagination.pageIndex}
            />
        </>
    );
};
