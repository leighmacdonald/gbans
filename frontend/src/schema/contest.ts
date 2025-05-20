import { z } from 'zod/v4';
import { schemaAsset } from './asset.ts';
import { schemaDateRange, schemaTimeStamped } from './chrono.ts';
import { PermissionLevelEnum } from './people.ts';

export const schemaContest = z
    .object({
        contest_id: z.string().optional(),
        deleted: z.boolean(),
        description: z.string(),
        down_votes: z.boolean(),
        hide_submissions: z.boolean(),
        max_submissions: z.number(),
        media_types: z.string(),
        min_permission_level: PermissionLevelEnum,
        num_entries: z.number(),
        public: z.boolean(),
        title: z.string(),
        voting: z.boolean()
    })
    .merge(schemaDateRange)
    .merge(schemaTimeStamped);

export type Contest = z.infer<typeof schemaContest>;

export const schemaContestEntry = z
    .object({
        contest_id: z.string(),
        contest_entry_id: z.string(),
        description: z.string(),
        asset_id: z.string(),
        steam_id: z.string(),
        placement: z.number(),
        personaname: z.string(),
        avatar_hash: z.string(),
        votes_up: z.number(),
        votes_down: z.number(),
        asset: schemaAsset
    })
    .merge(schemaTimeStamped);

export type ContestEntry = z.infer<typeof schemaContestEntry>;

export const schemaVoteResult = z.object({
    current_vote: z.enum(['up', 'down'])
});

export type VoteResult = z.infer<typeof schemaVoteResult>;
