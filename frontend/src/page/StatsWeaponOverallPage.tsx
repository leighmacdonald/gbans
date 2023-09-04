import React, { JSX, useEffect, useMemo, useState } from 'react';
import Grid from '@mui/material/Unstable_Grid2';
import { apiGetPlayerWeaponStats, PlayerWeaponStats, Weapon } from '../api';
import { LoadingSpinner } from '../component/LoadingSpinner';
import { ContainerWithHeader } from '../component/ContainerWithHeader';
import { LazyTable } from '../component/LazyTable';
import {
    compare,
    Order,
    RowsPerPage,
    stableSort
} from '../component/DataTable';
import Pagination from '@mui/material/Pagination';
import Stack from '@mui/material/Stack';
import InsightsIcon from '@mui/icons-material/Insights';
import { defaultFloatFmtPct, humanCount } from '../util/text';
import { useParams } from 'react-router';
import { PersonCell } from '../component/PersonCell';
import { fmtWhenGt } from '../component/PlayersOverallContainer';

interface WeaponStatsContainerProps {
    weapon_id: number;
}

const WeaponStatsContainer = ({ weapon_id }: WeaponStatsContainerProps) => {
    const [details, setDetails] = useState<PlayerWeaponStats[]>([]);
    const [loading, setLoading] = useState(true);
    const [sortOrder, setSortOrder] = useState<Order>('desc');
    const [sortColumn, setSortColumn] =
        useState<keyof PlayerWeaponStats>('kills');
    const [weapon, setWeapon] = useState<Weapon>();
    const [page, setPage] = useState(1);

    useEffect(() => {
        apiGetPlayerWeaponStats(weapon_id)
            .then((resp) => {
                if (resp.result) {
                    setDetails(resp.result.players);
                    setWeapon(resp.result.weapon);
                }
            })
            .finally(() => {
                setLoading(false);
            });
        // eslint-disable-next-line react-hooks/exhaustive-deps
    }, []);

    const rows = useMemo(() => {
        return stableSort(details ?? [], compare(sortOrder, sortColumn)).slice(
            (page - 1) * RowsPerPage.Hundred,
            (page - 1) * RowsPerPage.Hundred + RowsPerPage.Hundred
        );
    }, [details, page, sortColumn, sortOrder]);

    return (
        <ContainerWithHeader
            title={`Top 250 Weapon Users: ${weapon?.name}`}
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
                                    count={Math.ceil(
                                        details.length / RowsPerPage.Hundred
                                    )}
                                    showFirstButton
                                    showLastButton
                                    onChange={(_, newPage) => {
                                        setPage(newPage);
                                    }}
                                />
                            </Stack>
                            <LazyTable<PlayerWeaponStats>
                                columns={[
                                    {
                                        label: '#',
                                        sortable: true,
                                        sortKey: 'rank',
                                        tooltip: 'Overall Rank',
                                        align: 'center',
                                        renderer: (obj) => obj.rank
                                    },
                                    {
                                        label: 'Player Name',
                                        sortable: true,
                                        sortKey: 'personaname',
                                        tooltip: 'Player Name',
                                        align: 'left',
                                        renderer: (obj) => (
                                            <PersonCell
                                                avatar_hash={obj.avatar_hash}
                                                personaname={obj.personaname}
                                                steam_id={obj.steam_id}
                                            />
                                        )
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
                                        label: 'Dmg',
                                        sortable: true,
                                        sortKey: 'damage',
                                        tooltip: 'Total Damage',
                                        renderer: (obj) =>
                                            fmtWhenGt(obj.damage, humanCount)
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
                                        label: 'Hits',
                                        sortable: true,
                                        sortKey: 'hits',
                                        tooltip: 'Total Shots Landed',
                                        renderer: (obj) =>
                                            fmtWhenGt(obj.hits, humanCount)
                                    },
                                    {
                                        label: 'Acc%',
                                        sortable: false,
                                        virtual: true,
                                        virtualKey: 'accuracy',
                                        tooltip: 'Overall Accuracy',
                                        renderer: (obj) =>
                                            fmtWhenGt(obj.shots, () =>
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
                                        label: 'Bs',
                                        sortable: true,
                                        sortKey: 'backstabs',
                                        tooltip: 'Total Backstabs',
                                        renderer: (obj) =>
                                            fmtWhenGt(obj.backstabs, humanCount)
                                    },

                                    {
                                        label: 'Hs',
                                        sortable: true,
                                        sortKey: 'headshots',
                                        tooltip: 'Total Headshots',
                                        renderer: (obj) =>
                                            fmtWhenGt(obj.headshots, humanCount)
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

export const StatsWeaponOverallPage = (): JSX.Element => {
    const { weapon_id } = useParams();

    return (
        <Grid container spacing={2}>
            <Grid xs={12}>
                <WeaponStatsContainer weapon_id={parseInt(weapon_id ?? '0')} />
            </Grid>
        </Grid>
    );
};
