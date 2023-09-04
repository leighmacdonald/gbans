import React, { useEffect, useMemo, useState } from 'react';
import { apiError, apiResponse, WeaponsOverallResult } from '../api';
import { compare, Order, RowsPerPage, stableSort } from './DataTable';
import { useNavigate } from 'react-router-dom';
import { ContainerWithHeader } from './ContainerWithHeader';
import Grid from '@mui/material/Unstable_Grid2';
import { LoadingSpinner } from './LoadingSpinner';
import Stack from '@mui/material/Stack';
import Pagination from '@mui/material/Pagination';
import { LazyTable } from './LazyTable';
import Tooltip from '@mui/material/Tooltip';
import Button from '@mui/material/Button';
import Link from '@mui/material/Link';
import { defaultFloatFmtPct, humanCount } from '../util/text';
import { fmtWhenGt } from './PlayersOverallContainer';

interface WeaponsOverallContainerProps {
    fetchData: () => Promise<apiResponse<WeaponsOverallResult[]> & apiError>;
    title: string;
    icon: React.ReactNode;
}

export const WeaponsStatListContainer = ({
    fetchData,
    title,
    icon
}: WeaponsOverallContainerProps) => {
    const [details, setDetails] = useState<WeaponsOverallResult[]>([]);
    const [loading, setLoading] = useState(true);
    const [sortOrder, setSortOrder] = useState<Order>('desc');
    const [sortColumn, setSortColumn] =
        useState<keyof WeaponsOverallResult>('kills_pct');
    const [page, setPage] = useState(1);
    const navigate = useNavigate();

    useEffect(() => {
        fetchData()
            .then((resp) => {
                if (resp.result) {
                    setDetails(resp.result);
                }
            })
            .finally(() => {
                setLoading(false);
            });
    }, [fetchData]);

    const rows = useMemo(() => {
        return stableSort(details ?? [], compare(sortOrder, sortColumn)).slice(
            (page - 1) * RowsPerPage.TwentyFive,
            (page - 1) * RowsPerPage.TwentyFive + RowsPerPage.TwentyFive
        );
    }, [details, page, sortColumn, sortOrder]);

    return (
        <ContainerWithHeader title={title} iconLeft={icon}>
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
                                                    <Button
                                                        fullWidth
                                                        size={'small'}
                                                        variant={'text'}
                                                        component={Link}
                                                        href={`/stats/weapon/${obj.weapon_id}`}
                                                        onClick={(event) => {
                                                            event.preventDefault();
                                                            navigate(
                                                                `/stats/weapon/${obj.weapon_id}`
                                                            );
                                                        }}
                                                    >
                                                        {obj.name}
                                                    </Button>
                                                </Tooltip>
                                            );
                                        }
                                    },
                                    {
                                        label: 'Kills',
                                        sortable: true,
                                        sortKey: 'kills',
                                        tooltip: 'Total Kills',
                                        renderer: (obj) =>
                                            fmtWhenGt(obj.kills, humanCount)
                                    },
                                    {
                                        label: 'Kills%',
                                        sortable: true,
                                        sortKey: 'kills_pct',
                                        tooltip: 'Percentage Of Overall Kills',
                                        renderer: (obj) =>
                                            fmtWhenGt(
                                                obj.kills_pct,
                                                defaultFloatFmtPct
                                            )
                                    },
                                    {
                                        label: 'Shots',
                                        sortable: true,
                                        sortKey: 'shots',
                                        tooltip: 'Total Shots',
                                        renderer: (obj) =>
                                            fmtWhenGt(obj.shots, humanCount)
                                    },
                                    {
                                        label: 'Shots%',
                                        sortable: true,
                                        sortKey: 'shots_pct',
                                        tooltip: 'Shot Hit Percentage',
                                        renderer: (obj) =>
                                            fmtWhenGt(
                                                obj.shots_pct,
                                                defaultFloatFmtPct
                                            )
                                    },
                                    {
                                        label: 'Hits',
                                        sortable: true,
                                        sortKey: 'hits',
                                        tooltip: 'Total Hits',
                                        renderer: (obj) =>
                                            fmtWhenGt(obj.hits, humanCount)
                                    },
                                    {
                                        label: 'Hits%',
                                        sortable: true,
                                        sortKey: 'hits_pct',
                                        tooltip: 'Total Hits',
                                        renderer: (obj) =>
                                            fmtWhenGt(
                                                obj.hits_pct,
                                                defaultFloatFmtPct
                                            )
                                    },
                                    {
                                        label: 'Acc%',
                                        sortable: false,
                                        virtual: true,
                                        virtualKey: 'accuracy',
                                        tooltip: 'Overall Accuracy',
                                        renderer: (obj) =>
                                            fmtWhenGt(obj.shots_pct, () =>
                                                defaultFloatFmtPct(
                                                    (obj.hits / obj.shots) * 100
                                                )
                                            )
                                    },
                                    {
                                        label: 'As',
                                        sortable: true,
                                        sortKey: 'airshots',
                                        tooltip: 'Total Airshots',
                                        renderer: (obj) =>
                                            fmtWhenGt(obj.airshots, humanCount)
                                    },
                                    {
                                        label: 'As%',
                                        sortable: true,
                                        sortKey: 'airshots_pct',
                                        tooltip: 'Total Airshot Percentage',
                                        renderer: (obj) =>
                                            fmtWhenGt(
                                                obj.airshots_pct,
                                                defaultFloatFmtPct
                                            )
                                    },
                                    {
                                        label: 'Bs',
                                        sortable: true,
                                        sortKey: 'backstabs',
                                        tooltip: 'Total Backstabs',
                                        renderer: (obj) =>
                                            fmtWhenGt(obj.backstabs, humanCount)
                                    },
                                    {
                                        label: 'Bs%',
                                        sortable: true,
                                        sortKey: 'backstabs_pct',
                                        tooltip: 'Total Backstabs Percentage',
                                        renderer: (obj) =>
                                            fmtWhenGt(
                                                obj.backstabs_pct,
                                                defaultFloatFmtPct
                                            )
                                    },
                                    {
                                        label: 'Hs',
                                        sortable: true,
                                        sortKey: 'headshots',
                                        tooltip: 'Total Headshots',
                                        renderer: (obj) =>
                                            fmtWhenGt(obj.headshots, humanCount)
                                    },
                                    {
                                        label: 'Hs%',
                                        sortable: true,
                                        sortKey: 'headshots_pct',
                                        tooltip: 'Total Headshot Percentage',
                                        renderer: (obj) =>
                                            fmtWhenGt(
                                                obj.headshots_pct,
                                                defaultFloatFmtPct
                                            )
                                    },
                                    {
                                        label: 'Dmg',
                                        sortable: true,
                                        sortKey: 'damage',
                                        tooltip: 'Total Damage',
                                        renderer: (obj) =>
                                            fmtWhenGt(obj.damage, humanCount)
                                    },
                                    {
                                        label: 'Dmg%',
                                        sortable: true,
                                        sortKey: 'damage_pct',
                                        tooltip: 'Total Damage Percentage',
                                        renderer: (obj) =>
                                            fmtWhenGt(
                                                obj.damage_pct,
                                                defaultFloatFmtPct
                                            )
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
