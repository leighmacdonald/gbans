import { apiLogout } from "../../api";
import { accessTokenKey, logoutKey, profileKey } from "../../auth.tsx";
import { log } from "../errors.ts";

export const logoutFn = async () => {
	try {
		const ac = new AbortController();
		await apiLogout(ac.signal);
	} catch (error) {
		log(error);
	}
	localStorage.removeItem(profileKey);
	localStorage.removeItem(accessTokenKey);
	localStorage.setItem(logoutKey, Date.now().toString());
};
