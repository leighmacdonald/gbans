import { useEffect, useState } from 'react';
import { apiGetPersonSettings, PersonSettings } from '../api';
import { AppError } from '../error.tsx';

export const usePersonSettings = () => {
    const [data, setData] = useState<PersonSettings>();
    const [loading, setLoading] = useState(false);
    const [error, setError] = useState<AppError>();

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
