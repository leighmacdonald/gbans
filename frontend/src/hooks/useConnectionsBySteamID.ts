import { useEffect, useState } from 'react';
import {
    apiGetConnectionsSteam,
    PersonConnection,
    PersonConnectionSteamIDQuery
} from '../api';
import { logErr } from '../util/errors';

export const useConnectionsBySteamID = (opts: PersonConnectionSteamIDQuery) => {
    const [loading, setLoading] = useState(false);
    const [data, setData] = useState<PersonConnection[]>([]);
    const [count, setCount] = useState<number>(0);

    useEffect(() => {
        const abortController = new AbortController();

        setLoading(true);
        apiGetConnectionsSteam(
            {
                source_id: opts.source_id,
                desc: opts.desc,
                order_by: opts.order_by,
                limit: opts.limit,
                offset: opts.offset
            },
            abortController
        )
            .then((resp) => {
                setData(resp.data);
                setCount(resp.count);
            })
            .catch(logErr)
            .finally(() => {
                setLoading(false);
            });

        return () => abortController.abort();
    }, [opts.desc, opts.limit, opts.offset, opts.order_by, opts.source_id]);

    return { data, count, loading };
};
