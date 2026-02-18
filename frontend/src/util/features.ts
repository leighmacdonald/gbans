import { redirect } from "@tanstack/react-router";
import type { appInfoDetail } from "../schema/app.ts";
import { logErr } from "./errors.ts";

export const ensureFeatureEnabled = (featureName: keyof appInfoDetail, redirectTo: string = "/") => {
	if (!checkFeatureEnabled(featureName)) {
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
