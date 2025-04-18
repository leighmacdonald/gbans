import { LazyResult } from '../util/table.ts';
import { transformCreatedOnDate } from '../util/time.ts';
import { apiCall, QueryFilter } from './common.ts';

export type VoteResult = {
    server_id: number;
    server_name: string;
    match_id: string;
    source_id: string;
    source_name: string;
    source_avatar_hash: string;
    target_id: string;
    target_name: string;
    target_avatar_hash: string;
    success: boolean;
    valid: boolean;
    code: number;
    created_on: Date;
};

export type VoteQueryFilter = {
    source_id: string;
    target_id: string;
    success: number;
} & QueryFilter;

export const apiVotesQuery = async (opts: VoteQueryFilter, abortController?: AbortController) => {
    const resp = await apiCall<LazyResult<VoteResult>>('/api/votes', 'POST', opts, abortController);
    resp.data = resp.data.map(transformCreatedOnDate);

    return resp;
};
