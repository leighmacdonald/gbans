import { useEffect, useState } from 'react';
import { apiGetFilters, Filter, FiltersQueryFilter } from '../api/filters';
import { logErr } from '../util/errors';

export const useWordFilters = (opts: FiltersQueryFilter) => {
    const [data, setData] = useState<Filter[]>([]);
    const [count, setCount] = useState(0);
    const [loading, setLoading] = useState(false);
    const [error, setError] = useState();

    useEffect(() => {
        const abortController = new AbortController();
        setLoading(true);
        apiGetFilters(
            {
                order_by: opts.order_by,
                desc: opts.desc,
                limit: opts.limit,
                offset: opts.offset
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
    }, [opts.desc, opts.limit, opts.offset, opts.order_by]);

    return { data, count, loading, error };
};
