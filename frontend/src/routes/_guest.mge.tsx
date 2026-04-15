import { createFileRoute } from "@tanstack/react-router";
import { ensureFeatureEnabled } from "../util/features.ts";

export const Route = createFileRoute("/_guest/mge")({
	beforeLoad: ({ context }) => {
		ensureFeatureEnabled(context.appInfo.mgeEnabled);
	},
});
