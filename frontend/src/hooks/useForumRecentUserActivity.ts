import { useEffect, useState } from 'react';
import { ActiveUser, apiForumActiveUsers } from '../api/forum';
import { logErr } from '../util/errors';

export const useForumRecentUserActivity = () => {
    const [data, setData] = useState<ActiveUser[]>();
    const [loading, setLoading] = useState(false);
    const [error, setError] = useState();

    useEffect(() => {
        const abortController = new AbortController();
        setLoading(true);
        apiForumActiveUsers(abortController)
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
