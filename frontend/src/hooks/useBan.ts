import { useEffect, useState } from 'react';
import { apiGetBanSteam, SteamBanRecord } from '../api';
import { logErr } from '../util/errors';

export const useBan = (ban_id: number) => {
    const [loading, setLoading] = useState(false);
    const [data, setData] = useState<SteamBanRecord>();
    const [error, setError] = useState();

    useEffect(() => {
        const abortController = new AbortController();
        setLoading(true);
        apiGetBanSteam(ban_id, true)
            .then((ban) => {
                setData(ban);
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
