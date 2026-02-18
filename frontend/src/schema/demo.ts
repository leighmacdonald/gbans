import { z } from "zod/v4";

export const schemaDemoFile = z.object({
	demo_id: z.number(),
	server_id: z.number(),
	server_name_short: z.string(),
	server_name_long: z.string(),
	title: z.string(),
	created_on: z.date(),
	size: z.number(),
	downloads: z.number(),
	map_name: z.string(),
	archive: z.boolean(),
	stats: z.record(z.string(), z.unknown()),
	asset_id: z.string(),
});

export type DemoFile = z.infer<typeof schemaDemoFile>;
