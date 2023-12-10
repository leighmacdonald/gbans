import { useEffect, useState } from 'react';
import { APIError, apiGetPersonSettings, PersonSettings } from '../api';

export const usePersonSettings = () => {
    const [data, setData] = useState<PersonSettings>();
    const [loading, setLoading] = useState(false);
    const [error, setError] = useState<APIError>();

    useEffect(() => {
        const abortController = new AbortController();
        setLoading(true);
        apiGetPersonSettings(abortController)
            .then((resp) => {
                setData(resp ?? []);
            })
            .catch((reason) => {
                setError(reason);
            })
            .finally(() => {
                setLoading(false);
            });

        return () => abortController.abort();
    }, []);

    return { data, loading, error };
};
