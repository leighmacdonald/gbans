import { useEffect, useState } from 'react';
import { apiGetCIDRBlockLists, CIDRBlockLists } from '../api';
import { logErr } from '../util/errors';

export const useCIDRBlocks = () => {
    const [data, setData] = useState<CIDRBlockLists>();
    const [loading, setLoading] = useState(false);
    const [error, setError] = useState();

    useEffect(() => {
        const abortController = new AbortController();
        setLoading(true);
        apiGetCIDRBlockLists(abortController)
            .then((resp) => {
                setData(resp);
            })
            .catch((reason) => {
                setError(reason);
                logErr(reason);
            })
            .finally(() => {
                setLoading(false);
            });

        return () => abortController.abort();
    }, []);

    return { data, loading, error };
};
