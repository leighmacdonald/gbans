import { apiGetMapUsage, MapUseDetail } from '../api';
import React, { useEffect, useMemo, useState } from 'react';
import { PieChart } from '@mui/x-charts';
import { compare, Order, RowsPerPage, stableSort } from './DataTable';
import { ContainerWithHeader } from './ContainerWithHeader';
import MapIcon from '@mui/icons-material/Map';
import Grid from '@mui/material/Unstable_Grid2';
import Box from '@mui/material/Box';
import { LoadingSpinner } from './LoadingSpinner';
import Stack from '@mui/material/Stack';
import Pagination from '@mui/material/Pagination';
import { LazyTable } from './LazyTable';

interface MapUseChartProps {
    details: MapUseDetail[];
}

const MapUseChart = ({ details }: MapUseChartProps) => {
    const dataset = useMemo(() => {
        return details.map((d) => {
            return { id: d.map, label: d.map, value: d.percent };
        });
    }, [details]);

    return (
        <PieChart
            height={600}
            width={600}
            legend={{ hidden: true }}
            series={[
                {
                    data: dataset,
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
export const MapUsageContainer = () => {
    const [details, setDetails] = useState<MapUseDetail[]>([]);
    const [loading, setLoading] = useState(true);
    const [sortOrder, setSortOrder] = useState<Order>('desc');
    const [sortColumn, setSortColumn] = useState<keyof MapUseDetail>('percent');
    const [page, setPage] = useState(1);

    useEffect(() => {
        apiGetMapUsage()
            .then((resp) => {
                if (resp.result) {
                    setDetails(resp.result);
                }
            })
            .finally(() => {
                setLoading(false);
            });
    }, []);

    const rows = useMemo(() => {
        return stableSort(details, compare(sortOrder, sortColumn)).slice(
            (page - 1) * RowsPerPage.TwentyFive,
            (page - 1) * RowsPerPage.TwentyFive + RowsPerPage.TwentyFive
        );
    }, [details, page, sortColumn, sortOrder]);

    return (
        <ContainerWithHeader title={'Map Use Percent'} iconLeft={<MapIcon />}>
            <Grid container>
                <Grid md={6} xs={12}>
                    <Box
                        paddingLeft={10}
                        display="flex"
                        justifyContent="center"
                        alignItems="center"
                    >
                        {loading ? (
                            <LoadingSpinner />
                        ) : (
                            <MapUseChart details={details} />
                        )}
                    </Box>
                </Grid>
                <Grid md={6} xs={12}>
                    {loading ? (
                        <LoadingSpinner />
                    ) : (
                        <Stack>
                            <Stack direction={'row-reverse'}>
                                <Pagination
                                    page={page}
                                    count={Math.ceil(details.length / 25)}
                                    showFirstButton
                                    showLastButton
                                    onChange={(_, newPage) => {
                                        setPage(newPage);
                                    }}
                                />
                            </Stack>
                            <LazyTable<MapUseDetail>
                                columns={[
                                    {
                                        label: 'Map',
                                        sortable: true,
                                        sortKey: 'map',
                                        tooltip: 'Map'
                                    },
                                    // {
                                    //     label: 'Playtime',
                                    //     sortable: true,
                                    //     sortKey: 'playtime',
                                    //     tooltip: 'Total Playtime',
                                    //     renderer: (obj) => {
                                    //         return formatDistance(
                                    //             0,
                                    //             obj.playtime * 1000,
                                    //             {
                                    //                 includeSeconds: true
                                    //             }
                                    //         );
                                    //     }
                                    // },
                                    {
                                        label: 'Percent',
                                        sortable: true,
                                        sortKey: 'percent',
                                        tooltip:
                                            'Percentage of overall playtime',
                                        renderer: (obj) => {
                                            return (
                                                obj.percent.toFixed(2) + ' %'
                                            );
                                        }
                                    }
                                ]}
                                sortOrder={sortOrder}
                                sortColumn={sortColumn}
                                onSortColumnChanged={async (column) => {
                                    setSortColumn(column);
                                }}
                                onSortOrderChanged={async (direction) => {
                                    setSortOrder(direction);
                                }}
                                rows={rows}
                            />
                        </Stack>
                    )}
                </Grid>
            </Grid>
        </ContainerWithHeader>
    );
};
