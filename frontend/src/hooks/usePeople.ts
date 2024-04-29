import { useEffect, useState } from 'react';
import { apiSearchPeople, Person, PlayerQuery } from '../api';
import { logErr } from '../util/errors';

export const usePeople = (opts: PlayerQuery) => {
    const [data, setData] = useState<Person[]>([]);
    const [count, setCount] = useState(0);
    const [loading, setLoading] = useState(false);
    const [error, setError] = useState();

    useEffect(() => {
        const abortController = new AbortController();
        setLoading(true);
        apiSearchPeople(
            {
                personaname: opts.personaname,
                deleted: opts.deleted,
                desc: opts.desc,
                offset: opts.offset,
                limit: opts.limit,
                order_by: opts.order_by,
                target_id: opts.target_id,
                ip: opts.ip
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
    }, [opts.deleted, opts.desc, opts.ip, opts.limit, opts.offset, opts.order_by, opts.personaname, opts.target_id]);

    return { data, count, loading, error };
};
