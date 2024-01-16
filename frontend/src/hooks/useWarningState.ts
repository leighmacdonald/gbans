import { useEffect, useState } from 'react';
import { apiGetWarningState, warningState } from '../api/filters';
import { logErr } from '../util/errors';

export const useWarningState = () => {
    const [loading, setLoading] = useState(false);
    const [data, setData] = useState<warningState>({
        max_weight: 0,
        current: []
    });
    const [error, setError] = useState();

    useEffect(() => {
        const abortController = new AbortController();
        setLoading(true);
        apiGetWarningState(abortController)
            .then((state) => {
                setData(state);
            })
            .catch((reason) => {
                logErr(reason);
                setError(error);
            })
            .finally(() => {
                setLoading(false);
            });

        return () => abortController.abort();
    }, [error]);

    return { data, loading, error };
};
