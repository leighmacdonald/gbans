import { LazyResult } from '../util/table.ts';
import { apiCall, QueryFilter, transformCreatedOnDate } from './common.ts';

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
    success: boolean;
} & QueryFilter<VoteResult>;

export const apiVotesQuery = async (
    opts: VoteQueryFilter,
    abortController?: AbortController
) => {
    const resp = await apiCall<LazyResult<VoteResult>>(
        '/api/votes',
        'POST',
        opts,
        abortController
    );
    resp.data = resp.data.map(transformCreatedOnDate);

    return resp;
};
