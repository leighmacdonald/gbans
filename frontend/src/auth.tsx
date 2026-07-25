import { create } from "@bufbuild/protobuf";
import { timestampDate, timestampFromDate } from "@bufbuild/protobuf/wkt";
import { createClient } from "@connectrpc/connect";
import { type ReactNode, useCallback, useEffect, useState } from "react";
import { AuthContext } from "./contexts/AuthContext.tsx";
import { StorageType, useStorage } from "./hooks/useSessionStorage.tsx";
import { type PersonCore, PersonCoreSchema } from "./rpc/person/v1/person_core_pb.ts";
import { PersonService } from "./rpc/person/v1/person_pb.ts";
import { Privilege } from "./rpc/person/v1/privilege_pb.ts";
import { finalTransport } from "./transport.ts";
import { logErr } from "./util/errors.ts";
import { defaultAvatarHash } from "./util/strings.ts";
import { parseDateTime } from "./util/time.ts";
import type { Nullable } from "./util/types.ts";

export enum StorageKey {
	Profile = "profile",
	Logout = "logout",
}

type LocalStorageProfile = Nullable<
	Omit<Omit<PersonCore, "steamId">, "timeCreated"> & { steamId: string; timeCreated: Date }
>;

export function AuthProvider({ children }: { children: ReactNode }) {
	const [profile, setProfile] = useState<PersonCore>(loadProfile());

	const { setValue: setProfileValue, deleteValue: deleteProfileValue } = useStorage<LocalStorageProfile>(
		StorageKey.Profile,
		undefined,
		StorageType.Local,
	);

	useEffect(() => {
		const tryAuth = async () => {
			try {
				const personClient = createClient(PersonService, finalTransport);
				const data = await personClient.currentProfile({});
				if (data?.profile) {
					setProfileValue({
						...data.profile,
						steamId: data.profile.steamId.toString(),
						timeCreated: data.profile.timeCreated ? timestampDate(data.profile.timeCreated) : new Date(),
					});
					setProfile(data.profile);
				}
			} catch {
				// No valid cookie session
			}
		};

		if (profile.steamId === "") {
			tryAuth();
		}
	}, [setProfileValue, profile.steamId]);

	const login = useCallback(
		async (_token: string) => {
			const personClient = createClient(PersonService, finalTransport);
			const data = await personClient.currentProfile({});
			if (!data?.profile) {
				throw new Error("No profile");
			}
			setProfileValue({
				...data.profile,
				steamId: data.profile.steamId.toString(),
				timeCreated: data.profile.timeCreated ? timestampDate(data.profile.timeCreated) : new Date(),
			});
			setProfile(data.profile);
		},
		[setProfileValue],
	);

	const logout = useCallback(async () => {
		await fetch("/api/auth/logout", { method: "POST", credentials: "same-origin" });
		setProfile(defaultProfile);

		// Trigger logout on other tabs.
		localStorage.setItem(StorageKey.Logout, Date.now().toString());

		deleteProfileValue();
	}, [deleteProfileValue]);

	const isAuthenticated = () => {
		return profile.steamId !== "";
	};

	const permissionLevel = () => {
		return profile?.permissionLevel ?? Privilege.GUEST;
	};

	const hasPermission = (wantedLevel: Privilege) => {
		const currentLevel = permissionLevel();
		return currentLevel >= wantedLevel;
	};

	return (
		<AuthContext.Provider
			value={{
				profile,
				logout,
				isAuthenticated,
				permissionLevel,
				hasPermission,
				login,
			}}
		>
			{children}
		</AuthContext.Provider>
	);
}

const defaultProfile = create(PersonCoreSchema, {
	steamId: "",
	permissionLevel: Privilege.GUEST,
	avatarHash: defaultAvatarHash,
	name: "",
	banId: 0,
	discordId: "",
	timeCreated: undefined,
});

const loadProfile = (): PersonCore => {
	try {
		const userData = localStorage.getItem(StorageKey.Profile);
		if (!userData) {
			return defaultProfile;
		}

		const raw: LocalStorageProfile = JSON.parse(userData);
		if (!raw) {
			return defaultProfile;
		}

		return create(PersonCoreSchema, {
			...raw,
			steamId: raw.steamId,
			timeCreated: timestampFromDate(parseDateTime(raw.timeCreated)),
		});
	} catch (e) {
		logErr(e);
		return defaultProfile;
	}
};

export type AuthContextProps = {
	profile: PersonCore;
	login: (token: string) => Promise<void>;
	logout: () => Promise<void>;
	isAuthenticated: () => boolean;
	permissionLevel: () => Privilege;
	hasPermission: (level: Privilege) => boolean;
};
