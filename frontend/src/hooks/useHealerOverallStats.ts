import { useEffect, useMemo, useState } from 'react';
import {
    apiGetHealersOverall,
    HealingOverallResult,
    QueryFilter
} from '../api';
import { logErr } from '../util/errors';
import { compare, RowsPerPage, stableSort } from '../util/table.ts';

export const useHealerOverallStats = (
    opts: QueryFilter<HealingOverallResult>
) => {
    const [loading, setLoading] = useState(false);
    const [count, setCount] = useState<number>(0);
    const [allStats, setAllStats] = useState<HealingOverallResult[]>([]);

    useEffect(() => {
        const abortController = new AbortController();
        setLoading(true);
        apiGetHealersOverall()
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
            compare(opts.desc ? 'desc' : 'asc', opts.order_by ?? 'healing')
        ).slice(offset, offset + limit);
    }, [allStats, opts.desc, opts.limit, opts.offset, opts.order_by]);

    return { data, count, loading };
};
