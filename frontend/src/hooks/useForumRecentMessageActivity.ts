import { useEffect, useState } from 'react';
import { apiForumRecentActivity, ForumMessage } from '../api/forum';
import { logErr } from '../util/errors';

export const useForumRecentMessageActivity = () => {
    const [data, setData] = useState<ForumMessage[]>([]);
    const [loading, setLoading] = useState(false);
    const [error, setError] = useState();

    useEffect(() => {
        const abortController = new AbortController();
        setLoading(true);
        apiForumRecentActivity(abortController)
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
    }, []);

    return { data, loading, error };
};
