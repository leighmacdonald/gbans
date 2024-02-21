import { useEffect, useState } from 'react';
import { apiContests, Contest } from '../api';
import { logErr } from '../util/errors.ts';

export const useContests = () => {
    const [loading, setLoading] = useState(false);
    const [contests, setContests] = useState<Contest[]>([]);

    useEffect(() => {
        apiContests()
            .then((contests) => {
                setContests(contests.data);
            })
            .catch((e) => {
                setContests([]);
                logErr(e);
            })
            .finally(() => {
                setLoading(false);
            });
    }, []);

    return { loading, contests };
};
