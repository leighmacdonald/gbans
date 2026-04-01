import z from "zod/v4";
import { schemaQueryFilter } from "./query";

export const schemaMGEStat = z.object({
  stats_id: z.number(),
  rating: z.number(),
  steam_id: z.string(),
  personaname: z.string(),
  avatarhash: z.string(),
  wins: z.number(),
  losses: z.number(),
  lastplayed: z.date(),
  hitblip: z.number()
}).readonly()

export type MGEStat = z.infer<typeof schemaMGEStat>;


export const schemaQueryMGE = schemaQueryFilter.extend({
  steam_id: z.string().optional()
});

export type QueryMGE = z.infer<typeof schemaQueryMGE>;
