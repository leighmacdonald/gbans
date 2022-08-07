import Stack from '@mui/material/Stack';
import Typography from '@mui/material/Typography';
import { DataTable, UserTableProps } from './DataTable';
import { LoadingSpinner } from './LoadingSpinner';
import React, { useEffect, useMemo, useState } from 'react';
import { apiGetMatches, MatchesQueryOpts, MatchSummary } from '../api';
import { Link } from 'react-router-dom';

export interface MatchHistoryProps {
    opts: MatchesQueryOpts;
}

export const MatchHistory = ({ opts }: MatchHistoryProps) => {
    const [loading, setLoading] = useState<boolean>(true);
    const [matches, setMatches] = useState<MatchSummary[]>([]);

    useEffect(() => {
        apiGetMatches(opts)
            .then((matches) => {
                setMatches(matches || []);
            })
            .finally(() => {
                setLoading(false);
            });
    }, [opts]);

    const matchTableDef: UserTableProps<MatchSummary> = useMemo(() => {
        return {
            rowsPerPage: 25,
            columnOrder: [
                'match_id',
                'server_id',
                'map_name',
                'kills',
                'damage',
                'healing',
                'airshots',
                'created_on'
            ],
            defaultSortColumn: 'match_id',
            columns: [
                {
                    tooltip: 'Match ID',
                    label: '#',
                    sortKey: 'match_id',
                    sortType: 'number',
                    align: 'left',
                    renderer: (_, value) => {
                        return (
                            <Typography
                                variant={'button'}
                                component={Link}
                                to={`/log/${value}`}
                            >
                                {value as number}
                            </Typography>
                        );
                    }
                },
                {
                    tooltip: 'Server ID',
                    label: 'Srv',
                    sortKey: 'server_id'
                },
                {
                    tooltip: 'Map',
                    label: 'Map',
                    sortKey: 'map_name',
                    sortType: 'string'
                },
                {
                    tooltip: 'Kills',
                    label: 'Kills',
                    sortKey: 'kills'
                },
                {
                    tooltip: 'Damage',
                    label: 'Damage',
                    sortKey: 'damage'
                },
                {
                    tooltip: 'Healing',
                    label: 'Healing',
                    sortKey: 'healing'
                },
                {
                    tooltip: 'Airshots',
                    label: 'Airshots',
                    sortKey: 'airshots'
                },
                {
                    tooltip: 'Created',
                    label: 'Created',
                    sortKey: 'created_on',
                    sortType: 'date'
                }
            ],
            rows: matches
        };
    }, [matches]);

    if (loading) {
        return <LoadingSpinner />;
    }
    return (
        <Stack spacing={3} marginTop={3}>
            {matches.length == 0 && (
                <Typography variant={'caption'}>No results</Typography>
            )}
            {matches.length > 0 && <DataTable {...matchTableDef} />}
        </Stack>
    );
};
