import { useEffect, useState } from 'react';
import { apiGetForumOverview, ForumOverview } from '../api/forum';
import { logErr } from '../util/errors';

export const useForumOverview = () => {
    const [data, setData] = useState<ForumOverview>();
    const [loading, setLoading] = useState(false);
    const [error, setError] = useState();

    useEffect(() => {
        const abortController = new AbortController();
        setLoading(true);
        apiGetForumOverview(abortController)
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
