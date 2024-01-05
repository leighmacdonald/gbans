import { useEffect, useMemo, useState } from 'react';
import { apiGetPlayersOverall, PlayerWeaponStats, QueryFilter } from '../api';
import { RowsPerPage } from '../component/table/LazyTable';
import { compare, stableSort } from '../component/table/LazyTableSimple';
import { logErr } from '../util/errors';

export const usePlayersOverallStats = (
    opts: QueryFilter<PlayerWeaponStats>
) => {
    const [loading, setLoading] = useState(false);
    const [count, setCount] = useState<number>(0);
    const [allStats, setAllStats] = useState<PlayerWeaponStats[]>([]);

    useEffect(() => {
        const abortController = new AbortController();
        setLoading(true);
        apiGetPlayersOverall()
            .then((d) => {
                setAllStats(d.data);
                setCount(d.count);
            })
            .catch((e) => {
                logErr(e);
            })
            .finally(() => {
                setLoading(false);
            });
        return abortController.abort();
    }, []);

    const data = useMemo(() => {
        const limit = opts.limit ?? RowsPerPage.TwentyFive;
        const offset = opts.offset ?? 0;
        return stableSort(
            allStats,
            compare(opts.desc ? 'desc' : 'asc', opts.order_by ?? 'kills')
        ).slice(offset, offset + limit);
    }, [allStats, opts.desc, opts.limit, opts.offset, opts.order_by]);

    return { data, count, loading };
};
