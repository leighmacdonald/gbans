import { VoteQueryFilter, VoteResult } from '../schema/votes.ts';
import { LazyResult } from '../util/table.ts';
import { transformCreatedOnDate } from '../util/time.ts';
import { apiCall } from './common.ts';

export const apiVotesQuery = async (opts: VoteQueryFilter, abortController?: AbortController) => {
    const resp = await apiCall<LazyResult<VoteResult>>('/api/votes', 'POST', opts, abortController);
    resp.data = resp.data.map(transformCreatedOnDate);

    return resp;
};
