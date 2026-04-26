import { accessTokenKey, logoutKey, profileKey } from "../../auth.tsx";

export const logoutFn = async () => {
	localStorage.removeItem(profileKey);
	localStorage.removeItem(accessTokenKey);
	localStorage.setItem(logoutKey, Date.now().toString());
};
