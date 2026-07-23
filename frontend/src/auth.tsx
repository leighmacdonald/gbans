import { create } from "@bufbuild/protobuf";
import { EmptySchema, timestampDate, timestampFromDate } from "@bufbuild/protobuf/wkt";
import { createClient } from "@connectrpc/connect";
import { createConnectQueryKey } from "@connectrpc/connect-query";
import { type ReactNode, useCallback, useEffect, useState } from "react";
import { AuthContext } from "./contexts/AuthContext.tsx";
import { StorageType, useStorage } from "./hooks/useSessionStorage.tsx";
import { AuthService } from "./rpc/auth/v1/auth_pb.ts";
import { type PersonCore, PersonCoreSchema } from "./rpc/person/v1/person_core_pb.ts";
import { PersonService } from "./rpc/person/v1/person_pb.ts";
import { Privilege } from "./rpc/person/v1/privilege_pb.ts";
import { finalTransport, queryClient } from "./transport.ts";
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
	const authClient = createClient(AuthService, finalTransport);
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
		async (_token: string, opts: { onSuccess: () => void; onError: (error: Error) => void }) => {
			try {
				const personClient = createClient(PersonService, finalTransport);

				return queryClient
					.fetchQuery({
						queryKey: createConnectQueryKey({
							schema: PersonService,
							transport: finalTransport,
							cardinality: "finite",
						}),
						queryFn: async () => {
							return await personClient.currentProfile({});
						},
					})
					.then((data: CurrentProfileResponse) => {
						if (!data?.profile) {
							throw new Error("No profile");
						}

						setProfileValue({
							...data.profile,
							steamId: data.profile.steamId.toString(),
							timeCreated: profile.timeCreated ? timestampDate(profile.timeCreated) : new Date(),
						});
						setProfile(data.profile);
					})
					.then(opts.onSuccess)
					.catch(opts.onError);
			} catch (e) {
				opts.onError(e as Error);
				return Promise.reject(e);
			}
		},
		[setProfileValue, profile],
	);

	const logout = useCallback(async () => {
		await queryClient.fetchQuery({
			queryKey: createConnectQueryKey({
				schema: AuthService,
				transport: finalTransport,
				cardinality: "finite",
			}),
			queryFn: async () => {
				await authClient.logout(create(EmptySchema, {}));
				setProfile(defaultProfile);
			},
		});

		// Trigger logout on other tabs.
		localStorage.setItem(StorageKey.Logout, Date.now().toString());

		deleteProfileValue();
	}, [deleteProfileValue, authClient.logout]);

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
	login: (token: string, opts: { onSuccess: () => void; onError: (error: Error) => void }) => void;
	logout: () => Promise<void>;
	isAuthenticated: () => boolean;
	permissionLevel: () => Privilege;
	hasPermission: (level: Privilege) => boolean;
};
