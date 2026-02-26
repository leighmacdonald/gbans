import { queryOptions } from "@tanstack/react-query";
import { createFileRoute } from "@tanstack/react-router";
import { apiGetWikiPage } from "../api/wiki.ts";
import { WikiPage } from "../component/WikiPage.tsx";
import { PermissionLevel } from "../schema/people.ts";
import type { Page } from "../schema/wiki.ts";
import { logErr } from "../util/errors.ts";

export const Route = createFileRoute("/_guest/wiki/")({
	component: Wiki,
	head: () => ({
		meta: [{ name: "description", content: "Contests" }, { title: "Contests" }],
	}),
	loader: async ({ context, abortController }) => {
		const queryOpts = queryOptions({
			queryKey: ["wiki", { slug: "home" }],
			queryFn: async () => {
				try {
					return await apiGetWikiPage("home", abortController);
				} catch (e) {
					logErr(e);
					return {
						revision: 0,
						body_md: "",
						slug: "home",
						permission_level: PermissionLevel.Guest,
					} as Page;
				}
			},
		});

		return context.queryClient.fetchQuery(queryOpts);
	},
});

function Wiki() {
	return <WikiPage slug={"home"} path={"/_guest/wiki/"} />;
}
