import React, { JSX, useState } from 'react';
import { useParams } from 'react-router';
import InsightsIcon from '@mui/icons-material/Insights';
import Grid from '@mui/material/Unstable_Grid2';
import {
    apiGetPlayerWeaponStats,
    PlayerWeaponStatsResponse,
    Weapon
} from '../api';
import { ContainerWithHeader } from '../component/ContainerWithHeader';
import { PersonCell } from '../component/PersonCell';
import { fmtWhenGt } from '../component/PlayersOverallContainer';
import { RowsPerPage } from '../component/table/LazyTable';
import { LazyTableSimple } from '../component/table/LazyTableSimple';
import { defaultFloatFmtPct, humanCount } from '../util/text';

interface WeaponStatsContainerProps {
    weapon_id: number;
}

const WeaponStatsContainer = ({ weapon_id }: WeaponStatsContainerProps) => {
    const [weapon, setWeapon] = useState<Weapon>();

    const fetchData = async (): Promise<PlayerWeaponStatsResponse> => {
        const resp = await apiGetPlayerWeaponStats(weapon_id);
        setWeapon(resp.weapon);
        return resp;
    };

    return (
        <ContainerWithHeader
            title={`Top 250 Weapon Users: ${weapon?.name}`}
            iconLeft={<InsightsIcon />}
        >
            <Grid container>
                <Grid xs={12}>
                    <LazyTableSimple
                        fetchData={fetchData}
                        defaultSortColumn={'kills'}
                        defaultRowsPerPage={RowsPerPage.Fifty}
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
                                        defaultFloatFmtPct(obj.accuracy)
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
                    />
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
