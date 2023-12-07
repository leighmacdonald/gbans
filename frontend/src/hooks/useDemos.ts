import { useEffect, useState } from 'react';
import { apiGetDemos, DemoFile, DemoQueryFilter } from '../api';
import { logErr } from '../util/errors';

export const useDemos = (opts: DemoQueryFilter) => {
    const [data, setData] = useState<DemoFile[]>([]);
    const [count, setCount] = useState(0);
    const [loading, setLoading] = useState(false);
    const [error, setError] = useState();

    useEffect(() => {
        const abortController = new AbortController();
        setLoading(true);
        apiGetDemos(
            {
                limit: opts.limit,
                offset: opts.offset,
                order_by: opts.order_by,
                desc: opts.desc,
                steam_id: opts.steam_id,
                map_name: opts.map_name,
                server_ids: opts.server_ids
            },
            abortController
        )
            .then((resp) => {
                setData(resp.data);
                setCount(resp.count);
            })
            .catch((reason) => {
                setError(reason);
                logErr(reason);
            })
            .finally(() => {
                setLoading(false);
            });

        return () => abortController.abort();
    }, [
        opts.deleted,
        opts.desc,
        opts.limit,
        opts.map_name,
        opts.offset,
        opts.order_by,
        opts.server_ids,
        opts.steam_id
    ]);

    return { data, count, loading, error };
};
