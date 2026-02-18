import { queryOptions } from "@tanstack/react-query";
import { createFileRoute } from "@tanstack/react-router";
import { apiGetWikiPage } from "../api/wiki.ts";
import { ErrorDetails } from "../component/ErrorDetails.tsx";
import { Title } from "../component/Title.tsx";
import { WikiPage } from "../component/WikiPage.tsx";
import { AppError } from "../error.tsx";
import { PermissionLevel } from "../schema/people.ts";
import type { Page } from "../schema/wiki.ts";
import { toTitleCase } from "../util/text.tsx";

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
			return await context.queryClient.fetchQuery(queryOpts);
		} catch (e) {
			if (e instanceof AppError) {
				// Mostly meant for handling permission denied error
				throw e;
			}
			return {
				revision: 0,
				body_md: "",
				slug: slug,
				permission_level: PermissionLevel.Guest,
			} as Page;
		}
	},
	errorComponent: ({ error }) => {
		if (error instanceof AppError) {
			return <ErrorDetails error={error} />;
		}
		return <div>idk</div>;
	},
});

function Wiki() {
	const { slug } = Route.useParams();
	return (
		<>
			<Title>{toTitleCase(slug.replace(/-/g, " "))}</Title>
			<WikiPage slug={slug} path={"/_guest/wiki/$slug"} />
		</>
	);
}
