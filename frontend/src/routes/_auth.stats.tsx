import { createFileRoute } from "@tanstack/react-router";
import { ensureFeatureEnabled } from "../util/features.ts";

export const Route = createFileRoute("/_auth/stats")({
	beforeLoad: ({ context }) => {
		ensureFeatureEnabled(context.appInfo.stats_enabled);
	},
});
