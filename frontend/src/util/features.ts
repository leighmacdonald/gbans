import { redirect } from "@tanstack/react-router";

export const ensureFeatureEnabled = (enabled: boolean, redirectTo: string = "/") => {
	if (!enabled) {
		throw redirect({
			to: redirectTo,
		});
	}
};
