import { create } from "@bufbuild/protobuf";
import { timestampDate } from "@bufbuild/protobuf/wkt";
import { type ReactNode, useCallback } from "react";
import { AuthContext } from "./contexts/AuthContext.tsx";
import { type Person, PersonSchema } from "./rpc/person/v1/person_pb.ts";
import { Privilege } from "./rpc/person/v1/privilege_pb.ts";
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
	const login = useCallback(
		async (profile: Person) => {
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
			setProfile(create(PersonSchema, {}));
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
	profile: Person;
	login: (profile: Person) => void;
	logout: () => Promise<void>;
	isAuthenticated: () => boolean;
	permissionLevel: () => Privilege;
	hasPermission: (level: Privilege) => boolean;
};
