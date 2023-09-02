import React, { JSX, useEffect, useMemo, useState } from 'react';
import Grid from '@mui/material/Unstable_Grid2';
import {
    apiGetMapUsage,
    apiGetWeaponsOverall,
    MapUseDetail,
    WeaponsOverallResult
} from '../api';
import { LoadingSpinner } from '../component/LoadingSpinner';
import { PieChart } from '@mui/x-charts';
import { ContainerWithHeader } from '../component/ContainerWithHeader';
import Box from '@mui/material/Box';
import { LazyTable } from '../component/LazyTable';
import {
    compare,
    Order,
    RowsPerPage,
    stableSort
} from '../component/DataTable';
import MapIcon from '@mui/icons-material/Map';
import Pagination from '@mui/material/Pagination';
import Stack from '@mui/material/Stack';
import InsightsIcon from '@mui/icons-material/Insights';
import Typography from '@mui/material/Typography';
import Tooltip from '@mui/material/Tooltip';

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

const WeaponsOverallContainer = () => {
    const [details, setDetails] = useState<WeaponsOverallResult[]>([]);
    const [loading, setLoading] = useState(true);
    const [sortOrder, setSortOrder] = useState<Order>('desc');
    const [sortColumn, setSortColumn] =
        useState<keyof WeaponsOverallResult>('kills_pct');
    const [page, setPage] = useState(1);

    useEffect(() => {
        apiGetWeaponsOverall()
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
        return stableSort<WeaponsOverallResult>(
            details,
            compare(sortOrder, sortColumn)
        ).slice(
            (page - 1) * RowsPerPage.TwentyFive,
            (page - 1) * RowsPerPage.TwentyFive + RowsPerPage.TwentyFive
        );
    }, [details, page, sortColumn, sortOrder]);

    return (
        <ContainerWithHeader
            title={'Overall Weapon Info'}
            iconLeft={<InsightsIcon />}
        >
            <Grid container>
                <Grid xs={12}>
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
                            <LazyTable<WeaponsOverallResult>
                                columns={[
                                    {
                                        label: 'Weapon Name',
                                        sortable: true,
                                        sortKey: 'name',
                                        tooltip: 'Weapon Name',
                                        align: 'left',
                                        renderer: (obj) => {
                                            return (
                                                <Tooltip title={obj.key}>
                                                    <Typography
                                                        variant={'body2'}
                                                    >
                                                        {obj.name}
                                                    </Typography>
                                                </Tooltip>
                                            );
                                        }
                                    },
                                    {
                                        label: 'Kills',
                                        sortable: true,
                                        sortKey: 'kills',
                                        tooltip: 'Total Kills',
                                        renderer: (obj) => {
                                            return obj.kills;
                                        }
                                    },
                                    {
                                        label: 'Kills%',
                                        sortable: true,
                                        sortKey: 'kills_pct',
                                        tooltip: 'Percentage Of Overall Kills',
                                        renderer: (obj) => {
                                            return (
                                                obj.kills_pct.toFixed(2) + ' %'
                                            );
                                        }
                                    },
                                    {
                                        label: 'Shots',
                                        sortable: true,
                                        sortKey: 'shots',
                                        tooltip: 'Total Shots',
                                        renderer: (obj) => {
                                            return obj.shots > 0
                                                ? obj.shots
                                                : '';
                                        }
                                    },
                                    {
                                        label: 'Shots%',
                                        sortable: true,
                                        sortKey: 'shots_pct',
                                        tooltip: 'Total Kills',
                                        renderer: (obj) => {
                                            return obj.shots_pct > 0
                                                ? obj.shots_pct.toFixed(2) +
                                                      ' %'
                                                : '';
                                        }
                                    },
                                    {
                                        label: 'Hits',
                                        sortable: true,
                                        sortKey: 'hits',
                                        tooltip: 'Total Hits',
                                        renderer: (obj) => {
                                            return obj.hits > 0 ? obj.hits : '';
                                        }
                                    },
                                    {
                                        label: 'Hits%',
                                        sortable: true,
                                        sortKey: 'hits_pct',
                                        tooltip: 'Total Hits',
                                        renderer: (obj) => {
                                            return obj.hits_pct > 0
                                                ? obj.hits_pct.toFixed(2) + ' %'
                                                : '';
                                        }
                                    },
                                    {
                                        label: 'Acc%',
                                        sortable: false,
                                        virtual: true,
                                        virtualKey: 'accuracy',
                                        tooltip: 'Overall Accuracy',
                                        renderer: (obj) => {
                                            return obj.shots > 0
                                                ? (
                                                      (obj.hits / obj.shots) *
                                                      100
                                                  ).toFixed(2) + ' %'
                                                : '';
                                        }
                                    },
                                    {
                                        label: 'As',
                                        sortable: true,
                                        sortKey: 'airshots',
                                        tooltip: 'Total Airshots',
                                        renderer: (obj) => {
                                            return obj.airshots > 0
                                                ? obj.airshots
                                                : '';
                                        }
                                    },
                                    {
                                        label: 'As%',
                                        sortable: true,
                                        sortKey: 'airshots_pct',
                                        tooltip: 'Total Airshot Percentage',
                                        renderer: (obj) => {
                                            return obj.airshots_pct > 0
                                                ? obj.airshots_pct.toFixed(2) +
                                                      ' %'
                                                : '';
                                        }
                                    },
                                    {
                                        label: 'Bs',
                                        sortable: true,
                                        sortKey: 'backstabs',
                                        tooltip: 'Total Backstabs',
                                        renderer: (obj) => {
                                            return obj.backstabs > 0
                                                ? obj.backstabs
                                                : '';
                                        }
                                    },
                                    {
                                        label: 'Bs%',
                                        sortable: true,
                                        sortKey: 'backstabs_pct',
                                        tooltip: 'Total Backstabs Percentage',
                                        renderer: (obj) => {
                                            return obj.backstabs_pct > 0
                                                ? obj.backstabs_pct.toFixed(2) +
                                                      ' %'
                                                : '';
                                        }
                                    },
                                    {
                                        label: 'Hs',
                                        sortable: true,
                                        sortKey: 'headshots',
                                        tooltip: 'Total Headshots',
                                        renderer: (obj) => {
                                            return obj.headshots > 0
                                                ? obj.headshots
                                                : '';
                                        }
                                    },
                                    {
                                        label: 'Hs%',
                                        sortable: true,
                                        sortKey: 'headshots_pct',
                                        tooltip: 'Total Headshot Percentage',
                                        renderer: (obj) => {
                                            return obj.headshots_pct > 0
                                                ? obj.headshots_pct.toFixed(2) +
                                                      ' %'
                                                : '';
                                        }
                                    },
                                    {
                                        label: 'Dmg',
                                        sortable: true,
                                        sortKey: 'damage',
                                        tooltip: 'Total Damage',
                                        renderer: (obj) => {
                                            return obj.damage > 0
                                                ? obj.damage
                                                : '';
                                        }
                                    },
                                    {
                                        label: 'Dmg%',
                                        sortable: true,
                                        sortKey: 'damage_pct',
                                        tooltip: 'Total Damage Percentage',
                                        renderer: (obj) => {
                                            return obj.damage_pct > 0
                                                ? obj.damage_pct.toFixed(2) +
                                                      ' %'
                                                : '';
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

export const StatsPage = (): JSX.Element => {
    return (
        <Grid container spacing={2}>
            <Grid xs={12}>
                <WeaponsOverallContainer />
            </Grid>
            <Grid xs={12}>
                <MapUsageContainer />
            </Grid>
        </Grid>
    );
};
