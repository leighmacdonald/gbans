import { apiCall, PermissionLevel } from './common';

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
}

export const apiContestSave = async (contest: Contest) =>
    contest.contest_id == ''
        ? await apiCall<Contest, Contest>(`/api/contest`, 'POST', contest)
        : await apiCall<Contest, Contest>(
              `/api/contest/${contest.contest_id}`,
              'PUT',
              contest
          );
