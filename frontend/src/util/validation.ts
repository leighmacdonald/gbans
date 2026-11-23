import { z } from "zod/v4";

// Minimum valid SteamID64 (76561197960265728 is the first legacy SteamID2 mapping).
const MIN_STEAM_ID64 = 76561197960265729n;

export const steamID64Schema = z.string().refine((value) => {
	if (!/^\d+$/.test(value)) {
		return false;
	}
	try {
		return BigInt(value) >= MIN_STEAM_ID64;
	} catch {
		return false;
	}
}, "Enter a valid SteamID64");

/** Accepts an empty value, a plain IPv4 address, or an IPv4 address with a /0–/32 prefix. */
export const optionalCIDRSchema = z.string().refine((value) => {
	if (value === "") {
		return true;
	}

	const parts = value.split("/");
	if (parts.length > 2 || !z.ipv4().safeParse(parts[0]).success) {
		return false;
	}
	if (parts.length === 1) {
		return true;
	}

	const prefix = Number(parts[1]);

	return Number.isInteger(prefix) && prefix >= 0 && prefix <= 32;
}, "Enter a valid IPv4 address or CIDR");

export const futureDateSchema = z.date().refine((value) => value.getTime() > Date.now(), "Must be in the future");

/** Discord snowflake IDs are numeric; empty means "no channel configured". */
export const optionalDiscordIDSchema = z
	.string()
	.refine((value) => value === "" || /^\d+$/.test(value), "Enter a valid Discord ID");
