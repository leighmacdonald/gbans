import { useEffect, useState } from 'react';
import { apiContest, Contest } from '../api';
import { AppError } from '../error.tsx';

export const useContest = (contest_id?: string) => {
    const [loading, setLoading] = useState(false);
    const [contest, setContest] = useState<Contest>();
    const [error, setError] = useState<AppError>();

    useEffect(() => {
        if (!contest_id) {
            return;
        }

        apiContest(contest_id)
            .then((contest) => {
                setContest(contest);
            })
            .catch((reason) => {
                setError(reason as AppError);
            })
            .finally(() => {
                setLoading(false);
            });
    }, [contest_id]);

    return { loading, contest, error };
};
