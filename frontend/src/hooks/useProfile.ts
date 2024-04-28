import { useEffect, useState } from 'react';
import SteamID from 'steamid';
import { apiGetProfile, PlayerProfile } from '../api';
import { AppError, ErrorCode } from '../error.tsx';

export const useProfile = (steamId: string) => {
    const [data, setData] = useState<PlayerProfile>();
    const [loading, setLoading] = useState(false);
    const [error, setError] = useState<AppError>();

    useEffect(() => {
        const id = new SteamID(steamId);
        if (!id.isValidIndividual()) {
            setError(new AppError(ErrorCode.Unknown, 'Invalid Steam ID'));
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
