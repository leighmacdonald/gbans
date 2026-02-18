import { z } from "zod/v4";

export const schemaAppInfoDetail = z.object({
	site_name: z.string(),
	app_version: z.string(),
	link_id: z.string(),
	sentry_dns_web: z.string(),
	asset_url: z.string(),
	patreon_client_id: z.string(),
	patreon_enabled: z.boolean(),
	discord_client_id: z.string(),
	discord_enabled: z.boolean(),
	default_route: z.string(),
	news_enabled: z.boolean(),
	forums_enabled: z.boolean(),
	contests_enabled: z.boolean(),
	wiki_enabled: z.boolean(),
	stats_enabled: z.boolean(),
	servers_enabled: z.boolean(),
	reports_enabled: z.boolean(),
	chatlogs_enabled: z.boolean(),
	demos_enabled: z.boolean(),
	speedruns_enabled: z.boolean(),
	playerqueue_enabled: z.boolean(),
});

export type appInfoDetail = z.infer<typeof schemaAppInfoDetail>;
