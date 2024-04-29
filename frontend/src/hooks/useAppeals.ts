import { useEffect, useState } from 'react';
import { apiGetAppeals, AppealQueryFilter, SteamBanRecord } from '../api';
import { logErr } from '../util/errors';

export const useAppeals = (opts: AppealQueryFilter) => {
    const [loading, setLoading] = useState(false);
    const [appeals, setAppeals] = useState<SteamBanRecord[]>([]);
    const [count, setCount] = useState<number>(0);

    useEffect(() => {
        const abortController = new AbortController();

        setLoading(true);
        apiGetAppeals(
            {
                desc: opts.desc,
                order_by: opts.order_by,
                source_id: opts.source_id,
                target_id: opts.target_id,
                offset: opts.offset,
                limit: opts.limit,
                appeal_state: opts.appeal_state
            },
            abortController
        )
            .then((response) => {
                setAppeals(response.data);
                setCount(response.count);
            })
            .catch(logErr)
            .finally(() => setLoading(false));

        return () => abortController.abort();
    }, [opts.appeal_state, opts.desc, opts.limit, opts.offset, opts.order_by, opts.source_id, opts.target_id]);

    return { appeals, count, loading };
};
