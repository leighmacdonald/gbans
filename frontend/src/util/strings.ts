import type { Theme } from "@mui/material";
import { z } from "zod/v4";
import type { Asset } from "../rpc/asset/v1/asset_pb.ts";
import { BanType } from "../rpc/ban/v1/ban_pb.ts";
import { Detection } from "../rpc/anticheat/v1/anticheat_pb.ts";
import { ReportStatus } from "../rpc/ban/v1/report_pb.ts";
import type { DiscordProfile } from "../rpc/discord/oauth/v1/discord_pb.ts";
import type { Admin, Group, Override } from "../rpc/sourcemod/v1/sourcemod_pb";

const characters = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789";

export const randomStringAlphaNum = (length: number) => {
	let result = "";

	for (let i = 0; i < length; i++) {
		result += characters.charAt(Math.floor(Math.random() * characters.length));
	}

	return result;
};

export const cidrHostCount = (cidr: string): number => {
	if (!cidr.includes("/")) {
		return 0;
	}
	const mask = parseInt(cidr.split("/")[1], 10);
	return mask === 32 ? 1 : mask === 31 ? 2 : 2 ** (32 - mask) - 2;
};

export const EMPTY_UUID = "feb4bf16-7f55-4cb4-923c-4de69a093b79";

export const Flags = z.enum([
	"z",
	"a",
	"b",
	"c",
	"d",
	"e",
	"f",
	"g",
	"h",
	"i",
	"j",
	"k",
	"l",
	"m",
	"n",
	"o",
	"p",
	"q",
	"r",
	"s",
	"t",
]);

export const schemaFlags = z.object({
	a: z.boolean(),
	b: z.boolean(),
	c: z.boolean(),
	d: z.boolean(),
	e: z.boolean(),
	f: z.boolean(),
	g: z.boolean(),
	h: z.boolean(),
	i: z.boolean(),
	j: z.boolean(),
	k: z.boolean(),
	l: z.boolean(),
	m: z.boolean(),
	n: z.boolean(),
	o: z.boolean(),
	p: z.boolean(),
	q: z.boolean(),
	r: z.boolean(),
	s: z.boolean(),
	t: z.boolean(),
	z: z.boolean(),
});

export type Flags = z.infer<typeof Flags>;

export const hasSMFlag = (flag: Flags, entity?: Admin | Group | Override) => {
	return entity?.flags.includes(flag) ?? false;
};

export const cleanMapName = (name: string): string => {
	if (!name.startsWith("workshop/")) {
		return name;
	}
	const a = name.split("/");
	if (a.length !== 2) {
		return name;
	}
	const b = a[1].split(".ugc");
	if (a.length !== 2) {
		return name;
	}
	return b[0];
};

export const assetURL = (asset: Asset): string => `/asset/${asset.assetId}`;

export const discordAvatarURL = (user: DiscordProfile) => {
	return `https://cdn.discordapp.com/avatars/${user.id}/${user.avatar}.png`;
};

export const defaultAvatarHash = "fef49e7fa7e1997310d705b2a6158ff8dc1cdfeb";

const humanize = (count: number, thresh: number, dp = 1, units: string[]) => {
	let u = -1;
	const r = 10 ** dp;

	do {
		count /= thresh;
		++u;
	} while (Math.round(Math.abs(count) * r) / r >= thresh && u < units.length - 1);

	return `${count.toFixed(dp)}${units[u]}`;
};

export const humanFileSize = (bytes: number, si = false, dp = 1) => {
	const thresh = si ? 1000 : 1024;

	if (Math.abs(bytes) < thresh) {
		return `${bytes} B`;
	}

	const units = si
		? ["kB", "MB", "GB", "TB", "PB", "EB", "ZB", "YB"]
		: ["KiB", "MiB", "GiB", "TiB", "PiB", "EiB", "ZiB", "YiB"];
	return humanize(bytes, thresh, dp, units);
};

export const humanCount = (count: number, dp: number = 1): string => {
	if (Math.abs(count) < 1000) {
		return `${count}`;
	}
	return humanize(count, 1000, dp, ["K", "M", "B", "T", "Q"]);
};

export const defaultFloatFmtPct = (value: number) => `${value.toFixed(2)}%`;

export const defaultFloatFmt = (value: number) => value.toFixed(2);

type avatarSize = "small" | "medium" | "full";

export const avatarHashToURL = (hash?: string, size: avatarSize = "full") => {
	return `https://avatars.steamstatic.com/${hash ?? defaultAvatarHash}${size === "small" ? "" : `_${size}`}.jpg`;
};

export const toTitleCase = (str: string) =>
	str
		.toLowerCase()
		.replace(/S+/, "_")
		.split("_")
		.map((item) => (item === "unspecified" ? "" : item.replace(item.charAt(0), item.charAt(0).toUpperCase())))
		.join(" ");

export const reportStatusColour = (rs: ReportStatus, theme: Theme): string => {
	switch (rs) {
		case ReportStatus.NEED_MORE_INFO:
			return theme.palette.warning.main;
		case ReportStatus.CLOSED_WITHOUT_ACTION:
			return theme.palette.error.main;
		case ReportStatus.CLOSED_WITH_ACTION:
			return theme.palette.success.main;
		default:
			return theme.palette.info.main;
	}
};

// export const privilegeString = (privilege: Privilege) => {
// 	switch (privilege) {
// 		case Privilege.ADMIN:
// 			return "Admin";
// 		case Privilege.MODERATOR:
// 			return "Moderator";
// 		case Privilege.EDITOR:
// 			return "Editor";
// 		case Privilege.RESERVED:
// 			return "Reserved";
// 		case Privilege.USER:
// 			return "User";
// 		case Privilege.GUEST:
// 			return "Guest";
// 		case Privilege.BANNED_UNSPECIFIED:
// 			return "Banned";
// 		default:
// 			return "Unknown";
// 	}
// };

// export const appealStateString = (appealState: AppealState) => {
// 	switch (appealState) {
// 		case AppealState.DENIED:
// 			return "Denied";
// 		case AppealState.ACCEPTED:
// 			return "Accepted";
// 		case AppealState.REDUCED:
// 			return "Reduced";
// 		case AppealState.NO_APPEAL:
// 			return "No Appeal";
// 		case AppealState.OPEN_UNSPECIFIED:
// 			return "Open";
// 		default:
// 			return "Open";
// 	}
// };

export const reportStatusString = (reportStatus: ReportStatus) => {
	switch (reportStatus) {
		case ReportStatus.NEED_MORE_INFO:
			return "Need Info";
		case ReportStatus.CLOSED_WITHOUT_ACTION:
			return "Closed (No Action)";
		case ReportStatus.CLOSED_WITH_ACTION:
			return "Closed (Action)";
		case ReportStatus.OPENED_UNSPECIFIED:
			return "Opened";
		default:
			return "Opened";
	}
};

export const detectionString = (detection: Detection) => {
	switch (detection) {
		case Detection.AIM_SNAP:
			return "Aim Snap";
		case Detection.BHOP:
			return "BHop";
		case Detection.CHEAT_CVAR:
			return "Cheat CVar";
		case Detection.CMD_NUM_SPIKE:
			return "Cmd Num Spike";
		case Detection.EYE_ANGLES:
			return "Eye Angles";
		case Detection.INTERP:
			return "Interp";
		case Detection.INVALID_USER_CMD:
			return "Invalid User Cmd";
		case Detection.OOB_CVAR:
			return "OOB CVar";
		case Detection.SILENT_AIM:
			return "Silent Aim";
		case Detection.TOO_MANY_CONNECTIONS:
			return "Too Many Conn";
		case Detection.UNSPECIFIED:
			return "Unknown";
		default:
			return "Unknown";
	}
};

export const banTypeString = (banType: BanType) => {
	switch (banType) {
		case BanType.BANNED:
			return "Banned";
		case BanType.NETWORK:
			return "Network";
		case BanType.NO_COMM:
			return "Mute/Gag";
		default:
			return "OK";
	}
};

// export const banReasonString = (banReason: BanReason) => {
// 	switch (banReason) {
// 		case BanReason.BOT_HOST:
// 			return "Bot Host";
// 		case BanReason.CUSTOM:
// 			return "Custom";
// 		case BanReason.EXPLOITING:
// 			return "Exploiting";
// 		case BanReason.HARASSMENT:
// 			return "Harassment";
// 		case BanReason.EVADING:
// 			return "Evading";
// 		case BanReason.CHEATING:
// 			return "Cheating";
// 		case BanReason.EXTERNAL:
// 			return "External";
// 		case BanReason.ITEM_DESCRIPTIONS:
// 			return "Item Name/Description";
// 		case BanReason.LANGUAGE:
// 			return "Language";
// 		case BanReason.PROFILE:
// 			return "Profile";
// 		case BanReason.RACISM:
// 			return "Racism";
// 		case BanReason.SPAM:
// 			return "Spam";
// 		case BanReason.USERNAME:
// 			return "Username";
// 		case BanReason.WARNINGS_EXCEEDED:
// 			return "Warnings Exceeded";
// 		case BanReason.UNSPECIFIED:
// 			return "Unspecified";
// 		default:
// 			return "Unspecified";
// 	}
// };

type DeepNullifyStrings<T> = T extends string
	? string | undefined
	: T extends Array<infer U>
		? Array<DeepNullifyStrings<U>>
		: T extends object
			? { [K in keyof T]: DeepNullifyStrings<T[K]> }
			: T;

const isPlainObject = (value: unknown): value is Record<string, unknown> =>
	typeof value === "object" && value !== null && !Array.isArray(value);

const nullifyValue = <T>(value: T): DeepNullifyStrings<T> => {
	if (value === "" || value === null) {
		return undefined as unknown as DeepNullifyStrings<T>;
	}
	if (Array.isArray(value)) {
		return value.map((item) => nullifyValue(item)) as DeepNullifyStrings<T>;
	}
	if (isPlainObject(value)) {
		const entries = Object.entries(value).map(([key, v]) => [key, nullifyValue(v)] as const);
		return Object.fromEntries(entries) as DeepNullifyStrings<T>;
	}
	return value as DeepNullifyStrings<T>;
};

export const nullifyEmptyStrings = <T>(obj: T): DeepNullifyStrings<T> => {
	return nullifyValue(obj);
};
