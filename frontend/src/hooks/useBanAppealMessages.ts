import { useEffect, useState } from 'react';
import { apiGetBanMessages, BanAppealMessage } from '../api';
import { logErr } from '../util/errors';

export const useBanAppealMessages = (ban_id: number) => {
    const [loading, setLoading] = useState(false);
    const [data, setData] = useState<BanAppealMessage[]>([]);
    const [error, setError] = useState();

    useEffect(() => {
        const abortController = new AbortController();
        if (ban_id <= 0) {
            return;
        }

        setLoading(true);
        apiGetBanMessages(ban_id)
            .then((messages) => {
                setData(messages);
            })
            .catch((reason) => {
                logErr(reason);
                setError(error);
            })
            .finally(() => {
                setLoading(false);
            });

        return () => abortController.abort();
    }, [ban_id, error]);

    return { data, loading, error };
};
