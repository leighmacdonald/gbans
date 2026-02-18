import { createFileRoute } from "@tanstack/react-router";
import { ensureFeatureEnabled } from "../util/features.ts";

export const Route = createFileRoute("/_guest/wiki")({
	beforeLoad: () => {
		ensureFeatureEnabled("wiki_enabled");
	},
});
