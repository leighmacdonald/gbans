import { useEffect, useState } from 'react';
import { apiGetThread, ForumThread } from '../api/forum';
import { logErr } from '../util/errors';

export const useThread = (threadId: number) => {
    const [data, setData] = useState<ForumThread>();
    const [loading, setLoading] = useState(false);
    const [error, setError] = useState();

    useEffect(() => {
        const abortController = new AbortController();
        setLoading(true);
        apiGetThread(threadId, abortController)
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
    }, [threadId]);

    return { data, loading, error };
};
