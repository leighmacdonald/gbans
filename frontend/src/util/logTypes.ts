import type { Theme } from "@mui/material";
import { LogEventType } from "../rpc/servers/v1/servers_pb.ts";

export type LogCategory = "chat" | "kills" | "damage" | "players" | "world" | "server" | "votes" | "other";

export interface LogCategoryDef {
	id: LogCategory;
	label: string;
}

export const LOG_CATEGORIES: LogCategoryDef[] = [
	{ id: "chat", label: "Chat" },
	{ id: "kills", label: "Kills" },
	{ id: "damage", label: "Damage" },
	{ id: "players", label: "Players" },
	{ id: "world", label: "World" },
	{ id: "server", label: "Server" },
	{ id: "votes", label: "Votes" },
	{ id: "other", label: "Other" },
];

const categoryByType: Partial<Record<LogEventType, LogCategory>> = {
	[LogEventType.LOG_EVENT_TYPE_SAY]: "chat",
	[LogEventType.LOG_EVENT_TYPE_SAY_TEAM]: "chat",

	[LogEventType.LOG_EVENT_TYPE_KILLED]: "kills",
	[LogEventType.LOG_EVENT_TYPE_KILL_ASSIST]: "kills",
	[LogEventType.LOG_EVENT_TYPE_SUICIDE]: "kills",
	[LogEventType.LOG_EVENT_TYPE_DOMINATION]: "kills",
	[LogEventType.LOG_EVENT_TYPE_REVENGE]: "kills",
	[LogEventType.LOG_EVENT_TYPE_KILLED_CUSTOM]: "kills",
	[LogEventType.LOG_EVENT_TYPE_POINT_CAPTURED]: "kills",
	[LogEventType.LOG_EVENT_TYPE_CAPTURE_BLOCKED]: "kills",

	[LogEventType.LOG_EVENT_TYPE_SHOT_FIRED]: "damage",
	[LogEventType.LOG_EVENT_TYPE_SHOT_HIT]: "damage",
	[LogEventType.LOG_EVENT_TYPE_DAMAGE]: "damage",
	[LogEventType.LOG_EVENT_TYPE_PICKUP]: "damage",
	[LogEventType.LOG_EVENT_TYPE_EMPTY_UBER]: "damage",
	[LogEventType.LOG_EVENT_TYPE_MEDIC_DEATH]: "damage",
	[LogEventType.LOG_EVENT_TYPE_MEDIC_DEATH_EX]: "damage",
	[LogEventType.LOG_EVENT_TYPE_LOST_UBER_ADVANTAGE]: "damage",
	[LogEventType.LOG_EVENT_TYPE_CHARGE_READY]: "damage",
	[LogEventType.LOG_EVENT_TYPE_CHARGE_DEPLOYED]: "damage",
	[LogEventType.LOG_EVENT_TYPE_CHARGE_ENDED]: "damage",
	[LogEventType.LOG_EVENT_TYPE_HEALED]: "damage",
	[LogEventType.LOG_EVENT_TYPE_EXTINGUISHED]: "damage",
	[LogEventType.LOG_EVENT_TYPE_FIRST_HEAL_AFTER_SPAWN]: "damage",
	[LogEventType.LOG_EVENT_TYPE_JARATE_ATTACK]: "damage",
	[LogEventType.LOG_EVENT_TYPE_MILK_ATTACK]: "damage",
	[LogEventType.LOG_EVENT_TYPE_GAS_ATTACK]: "damage",

	[LogEventType.LOG_EVENT_TYPE_BUILT_OBJECT]: "players",
	[LogEventType.LOG_EVENT_TYPE_CARRY_OBJECT]: "players",
	[LogEventType.LOG_EVENT_TYPE_KILLED_OBJECT]: "players",
	[LogEventType.LOG_EVENT_TYPE_DETONATED_OBJECT]: "players",
	[LogEventType.LOG_EVENT_TYPE_DROP_OBJECT]: "players",
	[LogEventType.LOG_EVENT_TYPE_JOINED_TEAM]: "players",
	[LogEventType.LOG_EVENT_TYPE_CHANGE_CLASS]: "players",
	[LogEventType.LOG_EVENT_TYPE_SPAWNED_AS]: "players",
	[LogEventType.LOG_EVENT_TYPE_CONNECTED]: "players",
	[LogEventType.LOG_EVENT_TYPE_DISCONNECTED]: "players",
	[LogEventType.LOG_EVENT_TYPE_VALIDATED]: "players",
	[LogEventType.LOG_EVENT_TYPE_ENTERED]: "players",
	[LogEventType.LOG_EVENT_TYPE_STEAM_AUTH]: "players",

	[LogEventType.LOG_EVENT_TYPE_W_ROUND_OVERTIME]: "world",
	[LogEventType.LOG_EVENT_TYPE_W_ROUND_START]: "world",
	[LogEventType.LOG_EVENT_TYPE_W_ROUND_WIN]: "world",
	[LogEventType.LOG_EVENT_TYPE_W_ROUND_LEN]: "world",
	[LogEventType.LOG_EVENT_TYPE_W_TEAM_SCORE]: "world",
	[LogEventType.LOG_EVENT_TYPE_W_TEAM_FINAL_SCORE]: "world",
	[LogEventType.LOG_EVENT_TYPE_W_GAME_OVER]: "world",
	[LogEventType.LOG_EVENT_TYPE_W_PAUSED]: "world",
	[LogEventType.LOG_EVENT_TYPE_W_RESUMED]: "world",
	[LogEventType.LOG_EVENT_TYPE_W_MINI_ROUND_WIN]: "world",
	[LogEventType.LOG_EVENT_TYPE_W_MINI_ROUND_LEN]: "world",
	[LogEventType.LOG_EVENT_TYPE_W_MINI_ROUND_SELECTED]: "world",
	[LogEventType.LOG_EVENT_TYPE_W_MINI_ROUND_START]: "world",
	[LogEventType.LOG_EVENT_TYPE_W_ROUND_SETUP_BEGIN]: "world",
	[LogEventType.LOG_EVENT_TYPE_W_INTERMISSION_WIN_LIMIT]: "world",

	[LogEventType.LOG_EVENT_TYPE_LOG_START]: "server",
	[LogEventType.LOG_EVENT_TYPE_LOG_STOP]: "server",
	[LogEventType.LOG_EVENT_TYPE_CVAR]: "server",
	[LogEventType.LOG_EVENT_TYPE_RCON]: "server",
	[LogEventType.LOG_EVENT_TYPE_MAP_LOAD]: "server",
	[LogEventType.LOG_EVENT_TYPE_SERVER_CONFIG_EXEC]: "server",
	[LogEventType.LOG_EVENT_TYPE_MAP_STARTED]: "server",

	[LogEventType.LOG_EVENT_TYPE_VOTE_DETAILS]: "votes",
	[LogEventType.LOG_EVENT_TYPE_VOTE_SUCCESS]: "votes",
	[LogEventType.LOG_EVENT_TYPE_VOTE_FAILED]: "votes",
};

export const categoryForLogType = (eventType: LogEventType): LogCategory => categoryByType[eventType] ?? "other";

// Each category is mapped to a semantic palette channel of the active theme.
// Dark mode prefers the more vibrant main variant (the light variants look
// washed out on the dark panel) unless its main variant lacks contrast, and
// light mode prefers the darker variants for contrast on the light panel.
const categoryPalette: Record<LogCategory, (palette: Theme["palette"]) => string> = {
	chat: (palette) => (palette.mode === "dark" ? palette.info.main : palette.info.dark),
	kills: (palette) => (palette.mode === "dark" ? palette.error.main : palette.error.dark),
	damage: (palette) => (palette.mode === "dark" ? palette.warning.main : palette.warning.dark),
	players: (palette) => (palette.mode === "dark" ? palette.success.main : palette.success.dark),
	world: (palette) => (palette.mode === "dark" ? palette.secondary.light : palette.secondary.dark),
	server: (palette) => palette.text.secondary,
	votes: (palette) => (palette.mode === "dark" ? palette.warning.light : palette.warning.main),
	other: (palette) => (palette.mode === "dark" ? palette.common.white : palette.text.disabled),
};

export const categoryColour = (theme: Theme, category: LogCategory): string => categoryPalette[category](theme.palette);
