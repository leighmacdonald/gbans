import { apiCall, PermissionLevel } from './common';
import { useEffect, useState } from 'react';
import { logErr } from '../util/errors';
import { LazyResult } from './stats';
import { EmptyUUID } from './const';

export interface Contest {
    contest_id: string;
    title: string;
    description: string;
    public: boolean;
    date_start: Date;
    date_end: Date;
    max_submissions: number;
    media_types: string;
    deleted: boolean;
    voting: boolean;
    min_permission_level: PermissionLevel;
    down_votes: boolean;
    num_entries: number;
}

export const apiContestSave = async (contest: Contest) =>
    contest.contest_id == EmptyUUID
        ? await apiCall<Contest, Contest>(`/api/contest`, 'POST', contest)
        : await apiCall<Contest, Contest>(
              `/api/contest/${contest.contest_id}`,
              'PUT',
              contest
          );

export const apiContests = async () =>
    await apiCall<LazyResult<Contest>>(`/api/contests`, 'GET');

export const apiContest = async (contest_id: number) =>
    await apiCall<Contest>(`/api/contests/${contest_id}`, 'GET');

export const useContests = () => {
    const [loading, setLoading] = useState(false);
    const [contests, setContests] = useState<Contest[]>([]);

    useEffect(() => {
        apiContests()
            .then((contests) => {
                setContests(contests.data);
            })
            .catch(logErr)
            .finally(() => {
                setLoading(false);
            });
    }, []);

    return { loading, contests };
};

export const useContest = (context_id?: number) => {
    const [loading, setLoading] = useState(false);
    const [contest, setContest] = useState<Contest>();

    useEffect(() => {
        if (!context_id) {
            return;
        }

        apiContest(context_id)
            .then((contest) => {
                setContest(contest);
            })
            .catch(logErr)
            .finally(() => {
                setLoading(false);
            });
    }, [context_id]);

    return { loading, contest };
};
