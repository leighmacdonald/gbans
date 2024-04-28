import { useEffect, useState } from 'react';
import { apiForum, Forum } from '../api/forum';
import { AppError } from '../error.tsx';

export const useForum = (forumId: number) => {
    const [data, setData] = useState<Forum>();
    const [loading, setLoading] = useState(false);
    const [error, setError] = useState<AppError>();

    useEffect(() => {
        const abortController = new AbortController();
        setLoading(true);
        apiForum(forumId, abortController)
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
    }, [forumId]);

    return { data, loading, error };
};
