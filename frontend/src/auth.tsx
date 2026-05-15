import { create } from "@bufbuild/protobuf";
import { timestampDate, timestampFromDate } from "@bufbuild/protobuf/wkt";
import { createClient } from "@connectrpc/connect";
import { createConnectQueryKey } from "@connectrpc/connect-query";
import { type ReactNode, useCallback, useState } from "react";
import { AuthContext } from "./contexts/AuthContext.tsx";
import { StorageType, useStorage } from "./hooks/useSessionStorage.tsx";
import { AuthService } from "./rpc/auth/v1/auth_pb.ts";
import { type PersonCore, PersonCoreSchema } from "./rpc/person/v1/person_core_pb.ts";
import { PersonService } from "./rpc/person/v1/person_pb.ts";
import { Privilege } from "./rpc/person/v1/privilege_pb.ts";
import { finalTransport, queryClient } from "./transport.ts";
import { defaultAvatarHash } from "./util/strings.ts";
import { parseDateTime } from "./util/time.ts";
import type { Nullable } from "./util/types.ts";

export enum StorageKey {
	Token = "token",
	Profile = "profile",
	Logout = "logout",
}

type LocalStorageProfile = Nullable<
	Omit<Omit<PersonCore, "steamId">, "timeCreated"> & { steamId: string; timeCreated: Date }
>;

const loadToken = () => {
	try {
		const value = localStorage.getItem(StorageKey.Token);
		return { token: value ? value : "" };
	} catch {
		return { token: undefined };
	}
};

export function AuthProvider({ children }: { children: ReactNode }) {
	const authClient = createClient(AuthService, finalTransport);
	const [profile, setProfile] = useState<PersonCore>(loadProfile());

	const {
		value: tokenValue,
		setValue: setTokenValue,
		deleteValue: deleteTokenValue,
	} = useStorage<{ token?: string }>(StorageKey.Token, loadToken(), StorageType.Local);
	const { setValue: setProfileValue, deleteValue: deleteProfileValue } = useStorage<LocalStorageProfile>(
		StorageKey.Profile,
		undefined,
		StorageType.Local,
	);

	const login = useCallback(
		async (token: string) => {
			try {
				console.log(`Logging in... ${token}`);
				const personClient = createClient(PersonService, finalTransport);
				setTokenValue({ token: token });

				queryClient
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
					.then((data) => {
						if (!data?.profile) {
							throw "No profile";
						}

						setProfileValue({
							...data.profile,
							steamId: data.profile.steamId.toString(),
							timeCreated: profile.timeCreated ? timestampDate(profile.timeCreated) : new Date(),
						});
						setProfile(data.profile);
					});
			} catch (e) {
				console.log(e);
				throw "Failed to authenticate";
			}
		},
		[setProfileValue, profile, setTokenValue],
	);

	const logout = useCallback(async () => {
		await queryClient.fetchQuery({
			queryKey: createConnectQueryKey({
				schema: AuthService,
				transport: finalTransport,
				cardinality: "finite",
			}),
			queryFn: async () => {
				return await authClient.logout({});
			},
		});

		// Trigger logout on other tabs.
		localStorage.setItem(StorageKey.Logout, Date.now().toString());

		deleteProfileValue();
		deleteTokenValue();
	}, [deleteProfileValue, deleteTokenValue, authClient.logout]);

	const isAuthenticated = () => {
		return (tokenValue?.token ? Boolean(tokenValue.token?.length > 0) : false) && profile.steamId !== 0n;
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
	steamId: 0n,
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
			steamId: BigInt(raw.steamId),
			timeCreated: timestampFromDate(parseDateTime(raw.timeCreated)),
		});
	} catch (e) {
		console.log(e);
		return defaultProfile;
	}
};

export type AuthContextProps = {
	profile: PersonCore;
	login: (token: string, opts: { onSuccess?: () => void; onError?: () => void }) => void;
	logout: () => Promise<void>;
	isAuthenticated: () => boolean;
	permissionLevel: () => Privilege;
	hasPermission: (level: Privilege) => boolean;
};
