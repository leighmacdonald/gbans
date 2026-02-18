import { z } from "zod/v4";
import { schemaTimeStamped } from "./chrono.ts";

export const schemaCIDRBlockSource = z
	.object({
		cidr_block_source_id: z.number(),
		name: z.string(),
		url: z.string(),
		enabled: z.boolean(),
	})
	.merge(schemaTimeStamped);

export type CIDRBlockSource = z.infer<typeof schemaCIDRBlockSource>;

export const schemaWhitelistIP = z
	.object({
		cidr_block_whitelist_id: z.number(),
		address: z.cidrv4(),
	})
	.merge(schemaTimeStamped);

export type WhitelistIP = z.infer<typeof schemaWhitelistIP>;

export const schemaWhitelistSteam = z
	.object({
		steam_id: z.string(),
		personaname: z.string(),
		avatar_hash: z.string(),
	})
	.merge(schemaTimeStamped);

export type WhitelistSteam = z.infer<typeof schemaWhitelistSteam>;

export const schemaCIDRBlockCheckResponse = z.object({
	blocked: z.boolean(),
	source: z.string(),
});

export type CIDRBlockCheckResponse = z.infer<typeof schemaCIDRBlockCheckResponse>;
