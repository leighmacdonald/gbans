import { queryOptions } from "@tanstack/react-query";
import { createFileRoute } from "@tanstack/react-router";
import { apiGetWikiPage } from "../api/wiki.ts";
import { ErrorDetails } from "../component/ErrorDetails.tsx";
import { WikiPage } from "../component/WikiPage.tsx";
import { AppError } from "../error.tsx";
import { PermissionLevel } from "../schema/people.ts";

export const Route = createFileRoute("/_guest/wiki/$slug")({
	component: Wiki,

	loader: async ({ context, abortController, params }) => {
		const { slug } = params;
		const queryOpts = queryOptions({
			queryKey: ["wiki", { slug }],
			queryFn: async () => {
				return await apiGetWikiPage(slug, abortController);
			},
		});
		try {
			return { page: await context.queryClient.fetchQuery(queryOpts) };
		} catch (e) {
			if (e instanceof AppError) {
				// Mostly meant for handling permission denied error
				throw e;
			}
			return {
				page: {
					revision: 0,
					body_md: "",
					slug: slug,
					permission_level: PermissionLevel.Guest,
					created_on: new Date(),
					updated_on: new Date(),
				},
			};
		}
	},
	head: ({ match }) => ({
		meta: [{ name: "description", content: "Wiki" }, match.context.title("Wiki")],
	}),
	errorComponent: ({ error }) => {
		if (error instanceof AppError) {
			return <ErrorDetails error={error} />;
		}
		return <div>idk</div>;
	},
});

function Wiki() {
	const { slug } = Route.useParams();
	const { page } = Route.useLoaderData();
	const { appInfo } = Route.useRouteContext();

	return <WikiPage slug={slug} page={page} assetURL={appInfo.asset_url} />;
}
