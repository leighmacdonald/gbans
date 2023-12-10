import { useEffect, useState } from 'react';
import SteamID from 'steamid';
import { APIError, apiGetProfile, ErrorCode, PlayerProfile } from '../api';

export const useProfile = (steamId: string) => {
    const [data, setData] = useState<PlayerProfile>();
    const [loading, setLoading] = useState(false);
    const [error, setError] = useState<APIError>();

    useEffect(() => {
        const id = new SteamID(steamId);
        if (!id.isValidIndividual()) {
            setError(new APIError(ErrorCode.Unknown, 'Invalid Steam ID'));
            setLoading(false);
            return;
        }
        const abortController = new AbortController();
        setLoading(true);
        apiGetProfile(id.getSteamID64(), abortController)
            .then((resp) => {
                setData(resp);
            })
            .catch((reason) => {
                setError(reason);
            })
            .finally(() => {
                setLoading(false);
            });

        return () => abortController.abort();
    }, [steamId]);

    return { data, loading, error };
};
