import { z } from 'zod';
import { DurationEnum } from './bans.ts';
import { schemaTimeStamped } from './chrono.ts';
import { schemaUserProfile } from './people.ts';

export const schemaMapDetail = z
    .object({
        map_id: z.number(),
        map_name: z.string()
    })
    .merge(schemaTimeStamped);

export type MapDetail = z.infer<typeof schemaMapDetail>;

export const schemaSpeedrunParticipant = z.object({
    person: schemaUserProfile,
    round_id: z.number(),
    steam_id: z.string(),
    kills: z.number(),
    destructions: z.number(),
    duration: DurationEnum,
    persona_name: z.string(),
    avatar_hash: z.string()
});
export type SpeedrunParticipant = z.infer<typeof schemaSpeedrunParticipant>;

export const schemaSpeedrunPointCaptures = z.object({
    speedrun_id: z.number(),
    round_id: z.number(),
    players: z.array(schemaSpeedrunParticipant),
    duration: DurationEnum,
    point_name: z.string()
});
export type SpeedrunPointCaptures = z.infer<typeof schemaSpeedrunPointCaptures>;

export const schemaSpeedrunResult = z.object({
    speedrun_id: z.number(),
    server_id: z.number(),
    rank: z.number(),
    initial_rank: z.number(),
    map_detail: schemaMapDetail,
    point_captures: z.array(schemaSpeedrunPointCaptures),
    players: z.array(schemaSpeedrunPointCaptures),
    duration: DurationEnum,
    player_count: z.number(),
    bot_count: z.number(),
    created_on: z.date(),
    category: z.string()
});

export type SpeedrunResult = z.infer<typeof schemaSpeedrunResult>;

export const schemaSpeedrunMapOverview = z.object({
    speedrun_id: z.number(),
    server_id: z.number(),
    rank: z.number(),
    initial_rank: z.number(),
    map_detail: schemaMapDetail,
    duration: DurationEnum,
    player_count: z.number(),
    bot_count: z.number(),
    created_on: z.date(),
    category: z.string(),
    total_players: z.number()
});
export type SpeedrunMapOverview = z.infer<typeof schemaSpeedrunMapOverview>;
