import { useEffect, useState } from 'react';
import { apiVotesQuery, VoteQueryFilter, VoteResult } from '../api/votes.ts';
import { logErr } from '../util/errors.ts';
import { hookResult } from './hookResult.ts';

export const useVotes = (opts: VoteQueryFilter): hookResult<VoteResult[]> => {
    const [loading, setLoading] = useState(false);
    const [count, setCount] = useState<number>(0);
    const [data, setData] = useState<VoteResult[]>([]);

    useEffect(() => {
        setLoading(true);
        apiVotesQuery({
            limit: opts.limit,
            offset: opts.offset,
            order_by: opts.order_by,
            desc: opts.desc,
            source_id: opts.source_id,
            target_id: opts.target_id,
            success: opts.success
        })
            .then((resp) => {
                setData(resp.data || []);
                setCount(resp.count);
            })
            .catch((e) => {
                logErr(e);
            })
            .finally(() => {
                setLoading(false);
            });
    }, [opts.desc, opts.limit, opts.offset, opts.success, opts.order_by, opts.source_id, opts.target_id]);

    return { data, count, loading };
};
