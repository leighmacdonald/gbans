import { type ReactNode, useCallback } from "react";
import { defaultAvatarHash } from "./api";
import { AuthContext } from "./contexts/AuthContext.tsx";
import { logoutFn } from "./util/auth/logoutFn.ts";
import { logErr } from "./util/errors.ts";
import { type PersonCore, PersonCoreSchema } from "./rpc/person/v1/person_core_pb.ts";
import { create } from "@bufbuild/protobuf";
import { Privilege } from "./rpc/person/v1/privilege_pb.ts";
import { timestampDate } from "@bufbuild/protobuf/wkt";

export const accessTokenKey = "token";
export const profileKey = "profile";
export const logoutKey = "logout";

const saveProfile = (profile: PersonCore) => {
	const v = {
		...profile,
		steamId: profile.steamId.toString(),
		banId: profile.banId.toString(),
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
	profile: PersonCore;
	setProfile: (v?: PersonCore) => void;
}) {
	const login = useCallback(
		async (profile: PersonCore) => {
			saveProfile(profile);
			setProfile(profile);
		},
		[setProfile],
	);

	const logout = useCallback(async () => {
		try {
			await logoutFn();
		} catch (e) {
			logErr(`error logging out: ${e}`);
		} finally {
			setProfile(
				create(PersonCoreSchema, {
					steamId: 0n,
					permissionLevel: Privilege.GUEST,
					avatarHash: defaultAvatarHash,
				}),
			);
		}
	}, [setProfile]);

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
	profile: PersonCore;
	login: (profile: PersonCore) => void;
	logout: () => Promise<void>;
	isAuthenticated: () => boolean;
	permissionLevel: () => Privilege;
	hasPermission: (level: Privilege) => boolean;
};
