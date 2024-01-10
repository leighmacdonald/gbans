import { useEffect, useState } from 'react';
import { apiGetMessages, MessageQuery, PersonMessage } from '../api';
import { logErr } from '../util/errors';

export const useChatHistory = (opts: MessageQuery) => {
    const [data, setData] = useState<PersonMessage[]>([]);
    const [count, setCount] = useState(0);
    const [loading, setLoading] = useState(false);
    const [error, setError] = useState();

    useEffect(() => {
        const abortController = new AbortController();
        setLoading(true);
        apiGetMessages(
            {
                personaname: opts.personaname,
                deleted: opts.deleted,
                desc: opts.desc,
                offset: opts.offset,
                limit: opts.limit,
                order_by: opts.order_by,
                match_id: opts.match_id,
                date_start: opts.date_start,
                date_end: opts.date_end,
                server_id: opts.server_id,
                source_id: opts.source_id,
                query: opts.query
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
        opts.match_id,
        opts.limit,
        opts.offset,
        opts.order_by,
        opts.personaname,
        opts.date_start,
        opts.date_end,
        opts.server_id,
        opts.source_id,
        opts.query
    ]);

    return { data, count, loading, error };
};
