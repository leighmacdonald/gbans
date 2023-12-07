import { useEffect, useState } from 'react';
import {
    apiGetThreadMessages,
    ForumMessage,
    ThreadMessageQueryOpts
} from '../api/forum';
import { logErr } from '../util/errors';

export const useThreadMessages = (opts: ThreadMessageQueryOpts) => {
    const [data, setData] = useState<ForumMessage[]>([]);
    const [count, setCount] = useState(0);
    const [loading, setLoading] = useState(false);
    const [error, setError] = useState();

    useEffect(() => {
        const abortController = new AbortController();
        setLoading(true);
        apiGetThreadMessages(
            {
                forum_thread_id: opts.forum_thread_id,
                offset: opts.offset,
                limit: opts.limit,
                order_by: 'forum_message_id',
                desc: false
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
    }, [opts.forum_thread_id, opts.limit, opts.offset]);

    return { data, count, loading, error };
};
