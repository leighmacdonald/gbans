import { useEffect, useState } from 'react';
import { apiGetConnections, PersonConnection, ConnectionQuery } from '../api';
import { logErr } from '../util/errors';

export const useConnections = (opts: ConnectionQuery) => {
    const [loading, setLoading] = useState(false);
    const [data, setData] = useState<PersonConnection[]>([]);
    const [count, setCount] = useState<number>(0);

    useEffect(() => {
        if (!(opts.cidr != '' || opts.asn > 0 || opts.source_id != '')) {
            return;
        }
        const abortController = new AbortController();

        setLoading(true);
        apiGetConnections(
            {
                cidr: opts.cidr,
                source_id: opts.source_id,
                asn: opts.asn,
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
    }, [
        opts.desc,
        opts.limit,
        opts.offset,
        opts.order_by,
        opts.cidr,
        opts.asn,
        opts.source_id
    ]);

    return { data, count, loading };
};
