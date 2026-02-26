import { redirect } from "@tanstack/react-router";
import type { appInfoDetail } from "../schema/app.ts";
import { logErr } from "./errors.ts";

export const ensureFeatureEnabled = (enabled: boolean, redirectTo: string = "/") => {
	if (!enabled) {
		throw redirect({
			to: redirectTo,
		});
	}
};

export const checkFeatureEnabled = (featureName: keyof appInfoDetail) => {
	const item = localStorage.getItem("appInfo");
	if (!item) {
		return false;
	}
	try {
		return (JSON.parse(item) as appInfoDetail)[featureName];
	} catch (e) {
		logErr(e);
		return false;
	}
};
