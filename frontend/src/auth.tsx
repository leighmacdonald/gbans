import { timestampDate } from "@bufbuild/protobuf/wkt";
import { createClient } from "@connectrpc/connect";
import { type ReactNode, useCallback, useEffect, useState } from "react";
import { AuthContext } from "./contexts/AuthContext.tsx";
import { StorageType, useStorage } from "./hooks/useSessionStorage.tsx";
import type { PersonCore } from "./rpc/person/v1/person_core_pb.ts";
import { PersonService } from "./rpc/person/v1/person_pb.ts";
import type { Permission } from "./rpc/roles/v1/roles_pb.ts";
import { finalTransport } from "./transport.ts";
import { defaultProfile, type LocalStorageProfile, loadProfile, StorageKey } from "./util/profileStorage.ts";

export { loadProfile, StorageKey };

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
				// No valid cookie session — drop any stale cached profile.
				setProfile(defaultProfile);
				deleteProfileValue();
			}
		};

		if (profile.steamId === "") {
			tryAuth();
		}
	}, [setProfileValue, deleteProfileValue, profile.steamId]);

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

	const hasPermission = (permission: Permission) => {
		return (profile.permissions ?? []).includes(permission);
	};

	return (
		<AuthContext.Provider
			value={{
				profile,
				logout,
				isAuthenticated,
				hasPermission,
				login,
			}}
		>
			{children}
		</AuthContext.Provider>
	);
}

export type AuthContextProps = {
	profile: PersonCore;
	login: (token: string) => Promise<void>;
	logout: () => Promise<void>;
	isAuthenticated: () => boolean;
	hasPermission: (permission: Permission) => boolean;
};
