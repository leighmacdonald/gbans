import { create } from "@bufbuild/protobuf";
import { timestampFromDate } from "@bufbuild/protobuf/wkt";
import { type PersonCore, PersonCoreSchema } from "../rpc/person/v1/person_core_pb.ts";
import type { Permission } from "../rpc/roles/v1/roles_pb.ts";
import { logErr } from "./errors.ts";
import { defaultAvatarHash } from "./strings.ts";
import { parseDateTime } from "./time.ts";
import type { Nullable } from "./types.ts";

export enum StorageKey {
	Profile = "profile:v2",
	Logout = "logout",
}

export const LegacyProfileKey = "profile";

export type LocalStorageProfile = Nullable<
	Omit<Omit<PersonCore, "steamId">, "timeCreated"> & { steamId: string; timeCreated: Date }
>;

export const defaultProfile = create(PersonCoreSchema, {
	steamId: "",
	permissions: [],
	avatarHash: defaultAvatarHash,
	name: "",
	banId: 0,
	discordId: "",
	timeCreated: undefined,
});

export const loadProfile = (): PersonCore => {
	try {
		try {
			localStorage.removeItem(LegacyProfileKey);
		} catch {
			// Ignore storage errors when wiping legacy caches.
		}
		const userData = localStorage.getItem(StorageKey.Profile);
		if (!userData) {
			return defaultProfile;
		}

		const raw = JSON.parse(userData) as LocalStorageProfile & { permissions?: unknown };
		if (!raw || typeof raw !== "object") {
			try {
				localStorage.removeItem(StorageKey.Profile);
			} catch {
				// Ignore storage errors.
			}
			return defaultProfile;
		}

		if (typeof raw.steamId !== "string" || raw.steamId === "") {
			return defaultProfile;
		}

		const permissions = Array.isArray(raw.permissions)
			? raw.permissions.filter((p): p is Permission => typeof p === "number")
			: [];

		return create(PersonCoreSchema, {
			steamId: raw.steamId,
			permissions,
			name: typeof raw.name === "string" ? raw.name : "",
			avatarHash: typeof raw.avatarHash === "string" ? raw.avatarHash : defaultAvatarHash,
			discordId: typeof raw.discordId === "string" ? raw.discordId : "",
			patreonId: typeof raw.patreonId === "string" ? raw.patreonId : "",
			vacBans: typeof raw.vacBans === "number" ? raw.vacBans : 0,
			gameBans: typeof raw.gameBans === "number" ? raw.gameBans : 0,
			banId: typeof raw.banId === "number" ? raw.banId : 0,
			timeCreated: raw.timeCreated ? timestampFromDate(parseDateTime(raw.timeCreated)) : undefined,
		});
	} catch (e) {
		logErr(e);
		return defaultProfile;
	}
};
