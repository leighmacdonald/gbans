import { createFileRoute } from "@tanstack/react-router";
import { ensureFeatureEnabled } from "../util/features.ts";

export const Route = createFileRoute("/_guest/wiki")({
	beforeLoad: ({ context }) => {
		ensureFeatureEnabled(context.appInfo.wiki_enabled);
	},
	head: ({ match }) => ({
		meta: [{ name: "description", content: "Wiki" }, match.context.title("Wiki")],
	}),
});
