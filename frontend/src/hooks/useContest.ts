import { useEffect, useState } from 'react';
import { apiContest, APIError, Contest } from '../api';

export const useContest = (contest_id?: string) => {
    const [loading, setLoading] = useState(false);
    const [contest, setContest] = useState<Contest>();
    const [error, setError] = useState<APIError>();

    useEffect(() => {
        if (!contest_id) {
            return;
        }

        apiContest(contest_id)
            .then((contest) => {
                setContest(contest);
            })
            .catch((reason) => {
                setError(reason as APIError);
            })
            .finally(() => {
                setLoading(false);
            });
    }, [contest_id]);

    return { loading, contest, error };
};
