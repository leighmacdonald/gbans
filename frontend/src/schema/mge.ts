import z from "zod/v4";
import { schemaQueryFilter } from "./query";

export const schemaMGEHistory = z
	.object({
		duel_id: z.number(),
		winner: z.string(),
		winner_personaname: z.string(),
		winner_avatarhash: z.string(),
		winner2: z.string(),
		winner2_personaname: z.string(),
		winner2_avatarhash: z.string(),
		loser: z.string(),
		loser_personaname: z.string(),
		loser_avatarhash: z.string(),
		loser2: z.string(),
		loser2_personaname: z.string(),
		loser2_avatarhash: z.string(),
		winner_score: z.number(),
		loser_score: z.number(),
		winlimit: z.number(),
		game_time: z.date(),
		map_name: z.string(),
		arena_name: z.string(),
	})
	.readonly();

export type MGEHistory = z.infer<typeof schemaMGEHistory>;

export const schemaMGEStat = z
	.object({
		stats_id: z.number(),
		rating: z.number(),
		steam_id: z.string(),
		personaname: z.string(),
		avatarhash: z.string(),
		wins: z.number(),
		losses: z.number(),
		last_played: z.date(),
		hitblip: z.number(),
	})
	.readonly();

export type MGEStat = z.infer<typeof schemaMGEStat>;

export enum DuelMode {
	OneVsOne,
	TwoVsTwo,
}

export const schemaQueryMGE = schemaQueryFilter.extend({
	steam_id: z.string().optional(),
});

export type QueryMGE = z.infer<typeof schemaQueryMGE>;

export const schemaQueryMGEHistory = schemaQueryFilter.extend({
	winner: z.string().optional(),
	loser: z.string().optional(),
	winner2: z.string().optional(),
	loser2: z.string().optional(),
	mode: z.enum(DuelMode),
});

export type QueryMGEHistory = z.infer<typeof schemaQueryMGEHistory>;
