import { queryOptions } from "@tanstack/react-query";
import { createFileRoute } from "@tanstack/react-router";
import { apiGetWikiPage } from "../api/wiki.ts";
import { WikiPage } from "../component/WikiPage.tsx";
import { PermissionLevel } from "../schema/people.ts";
import { logErr } from "../util/errors.ts";

export const Route = createFileRoute("/_guest/wiki/")({
	component: Wiki,
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
					};
				}
			},
		});

		return { page: context.queryClient.fetchQuery(queryOpts), appInfo: context.appInfo };
	},
	head: ({ loaderData }) => ({
		meta: [{ name: "description", content: "Contests" }, { title: `Contests - ${loaderData?.appInfo.site_name}` }],
	}),
});

function Wiki() {
	const { appInfo } = Route.useLoaderData();
	return <WikiPage slug={"home"} path={"/_guest/wiki/"} assetURL={appInfo.asset_url} />;
}
