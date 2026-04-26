import { redirect } from "@tanstack/react-router";
import type { InfoResponse } from "../rpc/config/v1/config_pb.ts";
import { logErr } from "./errors.ts";

export const ensureFeatureEnabled = (enabled: boolean, redirectTo: string = "/") => {
	if (!enabled) {
		throw redirect({
			to: redirectTo,
		});
	}
};

export const checkFeatureEnabled = (featureName: keyof InfoResponse) => {
	const item = localStorage.getItem("appInfo");
	if (!item) {
		return false;
	}
	try {
		return (JSON.parse(item) as InfoResponse)[featureName];
	} catch (e) {
		logErr(e);
		return false;
	}
};
