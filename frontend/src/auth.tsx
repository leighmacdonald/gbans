import { create } from "@bufbuild/protobuf";
import { timestampDate } from "@bufbuild/protobuf/wkt";
import { createClient } from "@connectrpc/connect";
import { createConnectQueryKey } from "@connectrpc/connect-query";
import { useQueryClient } from "@tanstack/react-query";
import { type ReactNode, useCallback } from "react";
import { AuthContext } from "./contexts/AuthContext.tsx";
import { AuthService } from "./rpc/auth/v1/auth_pb.ts";
import { type Person, PersonSchema } from "./rpc/person/v1/person_pb.ts";
import { Privilege } from "./rpc/person/v1/privilege_pb.ts";
import { finalTransport } from "./transport.ts";
import { logoutFn } from "./util/auth/logoutFn.ts";
import { logErr } from "./util/errors.ts";

export const accessTokenKey = "token";
export const profileKey = "profile";
export const logoutKey = "logout";

const saveProfile = (profile: Person) => {
	const v = {
		...profile,
		timeCreated: profile.timeCreated ? timestampDate(profile.timeCreated) : new Date(),
	};
	console.log(v);
	localStorage.setItem(profileKey, JSON.stringify(v));
};
export function AuthProvider({
	children,
	profile,
	setProfile,
}: {
	children: ReactNode;
	profile: Person;
	setProfile: (v?: Person) => void;
}) {
	const queryClient = useQueryClient();
	const login = useCallback(
		async (profile: Person) => {
			saveProfile(profile);
			setProfile(profile);
		},
		[setProfile],
	);

	const logout = useCallback(async () => {
		try {
			const authClient = createClient(AuthService, finalTransport);

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
			await logoutFn();
		} catch (e) {
			logErr(`error logging out: ${e}`);
		} finally {
			setProfile(create(PersonSchema, {}));
		}
	}, [setProfile, queryClient.fetchQuery]);

	const isAuthenticated = () => {
		return Boolean(profile?.steamId ?? false);
	};

	const permissionLevel = () => {
		return profile?.permissionLevel ?? Privilege.GUEST;
	};

	const hasPermission = (wantedLevel: Privilege) => {
		const currentLevel = permissionLevel();
		return currentLevel >= wantedLevel;
	};

	// useEffect(() => {
	// 	const loadProfile = async () => {
	// 		try {
	// 			const token = readAccessToken();
	// 			if (!emptyOrNullString(token)) {
	// 				const ac = new AbortController();
	// 				await login(await apiGetCurrentProfile(ac.signal));
	// 			}
	// 		} catch (e) {
	// 			logErr(e);
	// 			await logout();
	// 		}
	// 	};
	// 	loadProfile().catch(logErr);
	// }, [login, logout]);

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

export type AuthContextProps = {
	profile: Person;
	login: (profile: Person) => void;
	logout: () => Promise<void>;
	isAuthenticated: () => boolean;
	permissionLevel: () => Privilege;
	hasPermission: (level: Privilege) => boolean;
};
