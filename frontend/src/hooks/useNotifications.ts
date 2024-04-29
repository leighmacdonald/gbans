import { useEffect, useState } from 'react';
import { apiGetNotifications, NotificationsQuery, UserNotification } from '../api';
import { logErr } from '../util/errors';

export const useNotifications = (opts: NotificationsQuery) => {
    const [data, setData] = useState<UserNotification[]>([]);
    const [count, setCount] = useState(0);
    const [loading, setLoading] = useState(false);
    const [error, setError] = useState();

    useEffect(() => {
        const abortController = new AbortController();
        setLoading(true);
        apiGetNotifications(
            {
                limit: opts.limit,
                offset: opts.offset,
                order_by: opts.order_by,
                desc: opts.desc
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
    }, [opts.deleted, opts.desc, opts.limit, opts.offset, opts.order_by]);

    return { data, count, loading, error };
};
