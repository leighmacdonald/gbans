import { createFileRoute } from "@tanstack/react-router";
import { ensureFeatureEnabled } from "../util/features.ts";

export const Route = createFileRoute("/_guest/wiki")({
	beforeLoad: ({ context }) => {
		ensureFeatureEnabled(context.appInfo.wiki_enabled);
	},
	loader: ({ context }) => ({
		appInfo: context.appInfo,
	}),
	head: ({ loaderData }) => ({
		meta: [{ name: "description", content: "Wiki" }, { title: `Wiki - ${loaderData?.appInfo.site_name}` }],
	}),
});
