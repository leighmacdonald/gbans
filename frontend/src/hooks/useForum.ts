import { useEffect, useState } from 'react';
import { APIError } from '../api';
import { apiForum, Forum } from '../api/forum';

export const useForum = (forumId: number) => {
    const [data, setData] = useState<Forum>();
    const [loading, setLoading] = useState(false);
    const [error, setError] = useState<APIError>();

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
