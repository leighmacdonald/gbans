import { useEffect, useState } from 'react';
import { apiForum, Forum } from '../api/forum';
import { logErr } from '../util/errors';

export const useForum = (forumId: number) => {
    const [data, setData] = useState<Forum>();
    const [loading, setLoading] = useState(false);
    const [error, setError] = useState();

    useEffect(() => {
        const abortController = new AbortController();
        setLoading(true);
        apiForum(forumId, abortController)
            .then((resp) => {
                setData(resp ?? []);
            })
            .catch((reason) => {
                setError(reason);
                logErr(reason);
            })
            .finally(() => {
                setLoading(false);
            });

        return () => abortController.abort();
    }, [forumId]);

    return { data, loading, error };
};
