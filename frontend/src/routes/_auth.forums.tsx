import { createFileRoute } from "@tanstack/react-router";
import { ensureFeatureEnabled } from "../util/features.ts";

export const Route = createFileRoute("/_auth/forums")({
	beforeLoad: ({ context }) => {
		ensureFeatureEnabled(context.appInfo.forums_enabled);
	},
});
