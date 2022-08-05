import React, { useEffect, useMemo, useState } from 'react';
import {
    ApiException,
    apiGetMatch,
    Match,
    MatchMedicSum,
    MatchPlayerSum,
    MatchTeamSum
} from '../api';
import { Link, useNavigate, useParams } from 'react-router-dom';
import { UserTableProps, DataTable } from '../component/DataTable';
import Typography from '@mui/material/Typography';
import Stack from '@mui/material/Stack';
import { useUserFlashCtx } from '../contexts/UserFlashCtx';
import { first } from 'lodash-es';
import { LoadingSpinner } from '../component/LoadingSpinner';
import Paper from '@mui/material/Paper';
import { Heading } from '../component/Heading';

export const MatchPage = (): JSX.Element => {
    const navigate = useNavigate();
    const [match, setMatch] = useState<Match>();
    const [loading, setLoading] = React.useState<boolean>(true);
    const { match_id } = useParams();
    const match_id_num = parseInt(match_id || 'x');
    const { sendFlash } = useUserFlashCtx();

    if (isNaN(match_id_num) || match_id_num <= 0) {
        sendFlash('error', 'Invalid match id');
        navigate('/404');
    }

    useEffect(() => {
        if (match_id_num > 0) {
            apiGetMatch(match_id_num)
                .then((resp) => {
                    setMatch(resp);
                })
                .catch((r: ApiException) => {
                    sendFlash(
                        'error',
                        r.resp.status == 404
                            ? 'Unknown match id'
                            : 'Internal error'
                    );
                    navigate('/404');
                    return;
                })
                .finally(() => {
                    setLoading(false);
                });
        }
    }, [match_id_num, navigate, sendFlash, setMatch]);

    const playerTableDef: UserTableProps<MatchPlayerSum> = useMemo(() => {
        return {
            rowsPerPage: 100,
            columnOrder: [
                'SteamId',
                'Kills',
                'Assists',
                'Deaths',
                'Damage',
                'KDRatio',
                'KADRatio',
                'DamageTaken',
                'Healing',
                'HealingTaken',
                'Airshots',
                'HeadShots',
                'BackStabs'
            ],
            defaultSortColumn: 'Kills',
            columns: [
                {
                    label: 'Profile',
                    sortKey: 'SteamId',
                    tooltip: 'Profile',
                    align: 'left',
                    renderer: (_, value) => {
                        const p = first(
                            (match?.Players || []).filter(
                                (p) => p.steam_id == value
                            )
                        );
                        return (
                            <Typography
                                variant={'button'}
                                component={Link}
                                to={`/profile/${p?.steam_id}`}
                            >
                                {p?.personaname ?? `${p?.steam_id}`}
                            </Typography>
                        );
                    }
                },
                {
                    label: 'K',
                    sortKey: 'Kills',
                    tooltip: 'Kills',
                    sortable: true,
                    sortType: 'number'
                },
                {
                    label: 'A',
                    sortKey: 'Assists',
                    tooltip: 'Assists',
                    sortable: true,
                    sortType: 'number'
                },
                {
                    label: 'D',
                    sortKey: 'Deaths',
                    tooltip: 'Deaths',
                    sortable: true,
                    sortType: 'number'
                },
                {
                    label: 'Dmg',
                    sortKey: 'Damage',
                    tooltip: 'Damage',
                    sortable: true,
                    sortType: 'number'
                },
                {
                    label: 'KD',
                    sortKey: 'KDRatio',
                    tooltip: 'Kills:Death ratio',
                    sortType: 'float',
                    sortable: true
                },
                {
                    label: 'KAD',
                    sortKey: 'KADRatio',
                    tooltip: 'Kills+Assists:Death ratio',
                    sortType: 'float',
                    sortable: true
                },
                {
                    label: 'DT',
                    sortKey: 'DamageTaken',
                    tooltip: 'DTaken',
                    sortable: true,
                    sortType: 'number'
                },
                {
                    label: 'Healing',
                    sortKey: 'Healing',
                    tooltip: 'Healing',
                    sortable: true,
                    sortType: 'number'
                },
                {
                    label: 'HT',
                    sortKey: 'HealingTaken',
                    tooltip: 'Healing Taken',
                    sortable: true,
                    sortType: 'number'
                },
                {
                    label: 'AS',
                    sortKey: 'Airshots',
                    tooltip: 'Airshots',
                    sortable: true,
                    sortType: 'number'
                },
                {
                    label: 'HS',
                    sortKey: 'HeadShots',
                    tooltip: 'Headshots',
                    sortable: true,
                    sortType: 'number'
                },
                {
                    label: 'BS',
                    sortKey: 'BackStabs',
                    tooltip: 'Backstabs',
                    sortable: true,
                    sortType: 'number'
                }
            ],
            rows: match?.PlayerSums || []
        };
    }, [match?.PlayerSums, match?.Players]);

    const medicTableDef: UserTableProps<MatchMedicSum> = useMemo(() => {
        return {
            rowsPerPage: 100,
            columnOrder: [
                'SteamId',
                'Healing',
                'Charges',
                'Drops',
                'NearFullChargeDeath'
            ],
            defaultSortColumn: 'Healing',
            columns: [
                {
                    label: 'Profile',
                    sortKey: 'SteamId',
                    tooltip: 'Profile',
                    sortable: true,
                    align: 'left',
                    renderer: (_, value) => {
                        const p = first(
                            (match?.Players || []).filter(
                                (p) => p.steam_id == value
                            )
                        );
                        return (
                            <Typography
                                variant={'button'}
                                component={Link}
                                to={`/profile/${p?.steam_id}`}
                            >
                                {p?.personaname ?? `${p?.steam_id}`}
                            </Typography>
                        );
                    }
                },
                {
                    label: 'Healing',
                    sortKey: 'Healing',
                    tooltip: 'Healing',
                    sortable: true,
                    sortType: 'number'
                },
                {
                    label: 'Charges',
                    sortKey: 'Charges',
                    tooltip: 'Uber Charges',
                    sortable: true,
                    sortType: 'number',
                    renderer: (value: { K: number } | unknown) => {
                        return value
                            ? Object.values(value as { K: number }).reduce(
                                  (sum, current) => sum + current,
                                  0
                              )
                            : 0;
                    }
                },
                {
                    label: 'Drops',
                    sortKey: 'Drops',
                    tooltip: 'Uber Drops',
                    sortable: true,
                    sortType: 'number'
                },
                {
                    label: 'Near Full Death',
                    sortKey: 'NearFullChargeDeath',
                    tooltip: 'Near full uber death',
                    sortable: true,
                    sortType: 'number'
                }
            ],
            rows: match?.MedicSums || []
        };
    }, [match?.MedicSums, match?.Players]);

    const teamTableDef: UserTableProps<MatchTeamSum> = useMemo(() => {
        return {
            rowsPerPage: 100,
            columnOrder: [
                'Team',
                'Kills',
                'Damage',
                'Caps',
                'Charges',
                'Drops',
                'MidFights'
            ],
            defaultSortColumn: 'Kills',
            columns: [
                {
                    label: 'Team',
                    sortKey: 'Team',
                    tooltip: 'Team',
                    align: 'left',
                    sortable: true,
                    sortType: 'number'
                },
                {
                    label: 'Kills',
                    sortKey: 'Kills',
                    tooltip: 'Total Kills',
                    sortable: true,
                    sortType: 'number'
                },
                {
                    label: 'Damage',
                    sortKey: 'Damage',
                    tooltip: 'Total Team Damage',
                    sortable: true,
                    sortType: 'number'
                },
                {
                    label: 'Caps',
                    sortKey: 'Caps',
                    tooltip: 'Total Captures',
                    sortable: true,
                    sortType: 'number'
                },
                {
                    label: 'Ubers',
                    sortKey: 'Charges',
                    tooltip: 'Total uber charges',
                    sortable: true,
                    sortType: 'number'
                },
                {
                    label: 'Drops',
                    sortKey: 'Drops',
                    tooltip: 'Total uber drops',
                    sortable: true,
                    sortType: 'number'
                },
                {
                    label: 'Mid',
                    sortKey: 'MidFights',
                    tooltip: 'Midfight wins',
                    sortable: true,
                    sortType: 'number'
                }
            ],
            rows: match?.TeamSums || []
        };
    }, [match?.TeamSums]);

    return (
        <Stack spacing={3} marginTop={3}>
            <Heading>
                {`Logs #${match?.MatchID} - ${match?.Title} - ${match?.MapName}`}
            </Heading>
            {!loading && match?.MatchID && (
                <>
                    <Paper>
                        <DataTable {...playerTableDef} />
                    </Paper>
                    <Paper>
                        <DataTable {...medicTableDef} />
                    </Paper>
                    <Paper>
                        <DataTable {...teamTableDef} />
                    </Paper>
                </>
            )}
            {loading && <LoadingSpinner />}
        </Stack>
    );
};
