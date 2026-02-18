import { createContext, useContext } from "react";
import type { appInfoDetail } from "../schema/app.ts";
import { noop } from "../util/lists.ts";

export type AppInfoCtx = {
	appInfo: appInfoDetail;
	setAppInfo: (appInfo: appInfoDetail) => void;
};

export const UseAppInfoCtx = createContext<AppInfoCtx>({
	setAppInfo: () => noop,
	appInfo: {
		app_version: "master",
		link_id: "",
		sentry_dns_web: "",
		site_name: "Loading",
		asset_url: "/assets",
		patreon_client_id: "",
		discord_client_id: "",
		patreon_enabled: false,
		discord_enabled: false,
		default_route: "/",
		forums_enabled: false,
		news_enabled: true,
		chatlogs_enabled: false,
		demos_enabled: false,
		contests_enabled: false,
		reports_enabled: false,
		servers_enabled: true,
		stats_enabled: false,
		wiki_enabled: false,
		speedruns_enabled: false,
		playerqueue_enabled: false,
	},
});

export const useAppInfoCtx = () => useContext(UseAppInfoCtx);
