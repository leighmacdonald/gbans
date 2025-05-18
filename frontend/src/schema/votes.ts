import { z } from 'zod';
import { schemaQueryFilter } from './query.ts';

export const schemaVoteResult = z.object({
    server_id: z.number(),
    server_name: z.string(),
    match_id: z.string(),
    source_id: z.string(),
    source_name: z.string(),
    source_avatar_hash: z.string(),
    target_id: z.string(),
    target_name: z.string(),
    target_avatar_hash: z.string(),
    success: z.boolean(),
    valid: z.boolean(),
    code: z.number(),
    created_on: z.date()
});

export type VoteResult = z.infer<typeof schemaVoteResult>;

export const schemaVoteQueryFilter = z
    .object({
        source_id: z.string(),
        target_id: z.string(),
        success: z.number()
    })
    .merge(schemaQueryFilter);

export type VoteQueryFilter = z.infer<typeof schemaVoteQueryFilter>;
