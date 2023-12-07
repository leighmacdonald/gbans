import { useEffect, useState } from 'react';
import { apiGetThreads, ForumThread, ThreadQueryOpts } from '../api/forum';

export const useThreads = (opts: ThreadQueryOpts) => {
    const [data, setData] = useState<ForumThread[]>([]);
    const [loading, setLoading] = useState(false);
    const [count, setCount] = useState(0);
    const [error, setError] = useState();

    useEffect(() => {
        const abortController = new AbortController();
        setLoading(true);
        apiGetThreads(
            {
                forum_id: opts.forum_id,
                offset: opts.offset,
                limit: opts.limit,
                order_by: 'updated_on',
                desc: true
            },
            abortController
        )
            .then((resp) => {
                setData(resp.data);
                setCount(resp.count);
            })
            .catch((reason) => {
                setError(reason);
            })
            .finally(() => {
                setLoading(false);
            });

        return () => abortController.abort();
    }, [opts.forum_id, opts.limit, opts.offset]);

    return { data, count, loading, error };
};
