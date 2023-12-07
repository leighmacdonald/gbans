import { useEffect, useState } from 'react';
import { apiGetServers, ServerSimple } from '../api';
import { logErr } from '../util/errors';
import { hookResult } from './hookResult';

export const useServers = (): hookResult<ServerSimple[]> => {
    const [loading, setLoading] = useState(false);
    const [count, setCount] = useState<number>(0);
    const [data, setData] = useState<ServerSimple[]>([]);

    useEffect(() => {
        const abortController = new AbortController();
        setLoading(true);
        apiGetServers(abortController)
            .then((resp) => {
                setData(
                    resp.sort((a: ServerSimple, b: ServerSimple) => {
                        return a.server_name.localeCompare(b.server_name);
                    })
                );
                setCount(resp.length);
            })
            .catch((e) => {
                logErr(e);
            })
            .finally(() => {
                setLoading(false);
            });

        return () => abortController.abort();
    }, []);

    return { data, count, loading };
};
