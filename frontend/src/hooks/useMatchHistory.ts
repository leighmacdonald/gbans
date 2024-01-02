import { useEffect, useState } from 'react';
import { apiGetMatches, MatchesQueryOpts, MatchSummary } from '../api';
import { logErr } from '../util/errors';

export const useMatchHistory = (opts: MatchesQueryOpts) => {
    const [data, setData] = useState<MatchSummary[]>([]);
    const [count, setCount] = useState(0);
    const [loading, setLoading] = useState(false);
    const [error, setError] = useState();

    useEffect(() => {
        setLoading(true);
        const abortController = new AbortController();
        apiGetMatches(
            {
                steam_id: opts.steam_id,
                limit: opts.limit,
                offset: opts.offset,
                order_by: opts.order_by,
                desc: opts.desc
            },
            abortController
        )
            .then((resp) => {
                setCount(resp.count);
                setData(resp.data);
            })
            .catch((e) => {
                logErr(e);
                setError(e);
            })
            .finally(() => {
                setLoading(false);
            });
        return () => abortController.abort();
    }, [opts.desc, opts.limit, opts.offset, opts.order_by, opts.steam_id]);

    return { data, count, loading, error };
};
