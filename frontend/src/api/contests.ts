import {
    apiCall,
    DateRange,
    PermissionLevel,
    transformDateRange
} from './common';
import { useEffect, useState } from 'react';
import { logErr } from '../util/errors';
import { LazyResult } from './stats';
import { EmptyUUID } from './const';

export interface Contest extends DateRange {
    contest_id: string;
    title: string;
    description: string;
    public: boolean;
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
        ? await apiCall<Contest, Contest>(`/api/contests`, 'POST', contest)
        : await apiCall<Contest, Contest>(
              `/api/contest/${contest.contest_id}`,
              'PUT',
              contest
          );

export const apiContests = async () => {
    const resp = await apiCall<LazyResult<Contest>>(`/api/contests`, 'GET');
    if (resp.data) {
        resp.data = resp.data.map(transformDateRange);
    }

    return resp;
};

export const apiContest = async (contest_id: string) =>
    await apiCall<Contest>(`/api/contests/${contest_id}`, 'GET');

export const apiContestDelete = async (contest_id: string) =>
    await apiCall<Contest>(`/api/contests/${contest_id}`, 'DELETE');

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

export const useContest = (contest_id?: string) => {
    const [loading, setLoading] = useState(false);
    const [contest, setContest] = useState<Contest>();

    useEffect(() => {
        if (!contest_id) {
            return;
        }

        apiContest(contest_id)
            .then((contest) => {
                setContest(contest);
            })
            .catch(logErr)
            .finally(() => {
                setLoading(false);
            });
    }, [contest_id]);

    return { loading, contest };
};
