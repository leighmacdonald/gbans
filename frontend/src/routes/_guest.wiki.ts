import { createFileRoute } from "@tanstack/react-router";
import { ensureFeatureEnabled } from "../util/features.ts";

export const Route = createFileRoute("/_guest/wiki")({
	beforeLoad: ({ context }) => {
		ensureFeatureEnabled(context.appInfo.wikiEnabled);
	},
	head: ({ match }) => ({
		meta: [{ name: "description", content: "Wiki" }, match.context.title("Wiki")],
	}),
});
