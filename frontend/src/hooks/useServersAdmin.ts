import { useEffect, useState } from 'react';
import { apiGetServersAdmin, Server, ServerQueryFilter } from '../api';
import { logErr } from '../util/errors';
import { hookResult } from './hookResult';

export const useServersAdmin = (
    opts: ServerQueryFilter
): hookResult<Server[]> => {
    const [loading, setLoading] = useState(false);
    const [count, setCount] = useState<number>(0);
    const [data, setData] = useState<Server[]>([]);

    useEffect(() => {
        const abortController = new AbortController();
        setLoading(true);
        apiGetServersAdmin(
            {
                limit: opts.limit,
                offset: opts.offset,
                order_by: opts.order_by,
                desc: opts.desc,
                deleted: opts.deleted,
                query: opts.query,
                include_disabled: opts.include_disabled
            },
            abortController
        )
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
        return () => abortController.abort();
    }, [
        opts.deleted,
        opts.desc,
        opts.include_disabled,
        opts.limit,
        opts.offset,
        opts.order_by,
        opts.query
    ]);

    return { data, count, loading };
};
