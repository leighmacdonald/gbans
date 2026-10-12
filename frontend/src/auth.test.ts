import { create } from "@bufbuild/protobuf";
import { beforeEach, expect, test, vi } from "vitest";
import { type PersonCore, PersonCoreSchema } from "./rpc/person/v1/person_core_pb.ts";
import { Permission } from "./rpc/roles/v1/roles_pb.ts";
import { hasPermission as hasPermissionUtil } from "./util/permissions.ts";
import { loadProfile, StorageKey } from "./util/profileStorage.ts";

const store = new Map<string, string>();

beforeEach(() => {
	store.clear();
	vi.stubGlobal("localStorage", {
		getItem: (key: string) => (store.has(key) ? store.get(key) : null),
		setItem: (key: string, value: string) => {
			store.set(key, value);
		},
		removeItem: (key: string) => {
			store.delete(key);
		},
		clear: () => {
			store.clear();
		},
	} satisfies Pick<Storage, "getItem" | "setItem" | "removeItem" | "clear">);
});

test("returns the default logged-out profile when nothing is stored", () => {
	const profile = loadProfile();
	expect(profile.steamId).toBe("");
	expect(profile.permissions).toEqual([]);
});

test("old cached profile without permissions does not crash", () => {
	store.set(
		StorageKey.Profile,
		JSON.stringify({
			steamId: "76561198000000000",
			name: "old-user",
		}),
	);
	const profile = loadProfile();
	expect(profile.steamId).toBe("76561198000000000");
	expect(profile.permissions).toEqual([]);
	expect(() => (profile.permissions ?? []).includes(Permission.BAN_READ)).not.toThrow();
});

test("null and undefined permissions default to an empty list", () => {
	for (const permissions of [null, undefined]) {
		store.set(StorageKey.Profile, JSON.stringify({ steamId: "76561198000000000", permissions }));
		const profile = loadProfile();
		expect(profile.permissions).toEqual([]);
	}
});

test("non-numeric permission entries are filtered out", () => {
	store.set(
		StorageKey.Profile,
		JSON.stringify({ steamId: "76561198000000000", permissions: [Permission.BAN_READ, "nope", null] }),
	);
	const profile = loadProfile();
	expect(profile.permissions).toEqual([Permission.BAN_READ]);
});

test("garbage JSON returns the default profile", () => {
	store.set(StorageKey.Profile, "{not-json");
	const profile = loadProfile();
	expect(profile.steamId).toBe("");
	expect(profile.permissions).toEqual([]);
});

test("legacy v1 cache key is wiped on load", () => {
	store.set("profile", JSON.stringify({ steamId: "76561198000000000" }));
	loadProfile();
	expect(store.has("profile")).toBe(false);
});

test("permissions util tolerates missing permissions", () => {
	const person = create(PersonCoreSchema, { steamId: "76561198000000000" }) as PersonCore;
	(person as unknown as { permissions: undefined }).permissions = undefined;
	expect(hasPermissionUtil(person, Permission.BAN_READ)).toBe(false);
});
