import { useEffect, useState } from 'react';
import { apiGetBansCIDR, BanCIDRQueryFilter, CIDRBanRecord } from '../api';
import { logErr } from '../util/errors';

export const useBansCIDR = (opts: BanCIDRQueryFilter) => {
    const [loading, setLoading] = useState(false);
    const [count, setCount] = useState<number>(0);
    const [data, setData] = useState<CIDRBanRecord[]>([]);
    const [error, setError] = useState();

    useEffect(() => {
        const abortController = new AbortController();
        setLoading(true);
        apiGetBansCIDR(
            {
                limit: opts.limit,
                offset: opts.offset,
                order_by: opts.order_by,
                desc: opts.desc,
                source_id: opts.source_id,
                target_id: opts.target_id,
                appeal_state: opts.appeal_state,
                deleted: opts.deleted,
                ip: opts.ip
            },
            abortController
        )
            .then((bans) => {
                setData(bans.data);
                setCount(bans.count);
            })
            .catch((reason) => {
                logErr(reason);
                setError(error);
            })
            .finally(() => {
                setLoading(false);
            });

        return () => abortController.abort();
    }, [
        opts.limit,
        opts.offset,
        opts.order_by,
        opts.desc,
        opts.source_id,
        opts.target_id,
        opts.appeal_state,
        opts.deleted,
        error,
        opts.ip
    ]);

    return { data, count, loading, error };
};
