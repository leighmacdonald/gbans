import { useEffect, useState } from 'react';
import { apiGetNewsAll, NewsEntry } from '../api/news';
import { logErr } from '../util/errors';

export const useNews = () => {
    const [data, setData] = useState<NewsEntry[]>([]);
    const [count, setCount] = useState(0);
    const [loading, setLoading] = useState(false);
    const [error, setError] = useState();

    useEffect(() => {
        const abortController = new AbortController();
        setLoading(true);
        apiGetNewsAll(abortController)
            .then((resp) => {
                setData(resp);
                setCount(resp.length);
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

    return { data, count, loading, error };
};
