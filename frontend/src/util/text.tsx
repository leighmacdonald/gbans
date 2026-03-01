import { defaultAvatarHash } from "../api";
import type { Person } from "../schema/people.ts";

export const filterPerson = (people: Person[], query: string): Person[] => {
	return people.filter((friend) => {
		if (friend.persona_name.toLowerCase().includes(query)) {
			return true;
		} else if (friend.steam_id === query) {
			return true;
		}
		// TODO convert steamids from other formats to query
		return false;
	});
};

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
		.split(" ")
		.map((item) => item.replace(item.charAt(0), item.charAt(0).toUpperCase()))
		.join(" ");
