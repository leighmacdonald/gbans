import { useEffect, useState } from 'react';
import { apiGetNetworkDetails, IPQuery, NetworkDetails } from '../api';
import { logErr } from '../util/errors';

export const useNetworkQuery = (opts: IPQuery) => {
    const [loading, setLoading] = useState(false);
    const [data, setData] = useState<NetworkDetails>();

    useEffect(() => {
        if (opts.ip == '') {
            return;
        }
        const abortController = new AbortController();

        setLoading(true);
        apiGetNetworkDetails(opts, abortController)
            .then((resp) => {
                setData(resp);
            })
            .catch(logErr)
            .finally(() => {
                setLoading(false);
            });

        return () => abortController.abort();
    }, [opts.ip]);

    return { data, loading };
};
