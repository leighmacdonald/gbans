import { queryOptions } from "@tanstack/react-query";
import { createFileRoute } from "@tanstack/react-router";
import { apiGetWikiPage } from "../api/wiki.ts";
import { WikiPage } from "../component/WikiPage.tsx";
import { PermissionLevel } from "../schema/people.ts";

export const Route = createFileRoute("/_guest/wiki/")({
	component: Wiki,
	loader: async ({ context, abortController }) => {
		const queryOpts = queryOptions({
			queryKey: ["wiki", { slug: "home" }],
			queryFn: async () => {
				try {
					return await apiGetWikiPage("home", abortController);
				} catch {
					return {
						revision: 0,
						body_md: "",
						slug: "home",
						permission_level: PermissionLevel.Guest,
						created_on: new Date(),
						updated_on: new Date(),
					};
				}
			},
		});
		const page = await context.queryClient.fetchQuery(queryOpts);
		return { page };
	},
	head: ({ match, loaderData }) => {
		return {
			meta: [{ name: "description", content: "Wiki" }, match.context.title(loaderData?.page.slug ?? "Home")],
		};
	},
});

function Wiki() {
	const { appInfo } = Route.useRouteContext();
	const { page } = Route.useLoaderData();
	return <WikiPage page={page} slug="home" assetURL={appInfo.asset_url} />;
}
