import { createFileRoute } from "@tanstack/react-router";
import { WikiPage } from "../component/WikiPage.tsx";
import { useSuspenseQuery } from "@connectrpc/connect-query";
import { get } from "../rpc/wiki/v1/wiki-WikiService_connectquery.ts";
import type { Wiki } from "../rpc/wiki/v1/wiki_pb.ts";

export const Route = createFileRoute("/_guest/wiki/")({
	component: Wiki,
	head: ({ match }) => {
		// TODO set title to slug
		return {
			meta: [{ name: "description", content: "Wiki" }, match.context.title("Wiki")],
		};
	},
});

function Wiki() {
	const { appInfo } = Route.useRouteContext();
	const { data: page } = useSuspenseQuery(get, { slug: "home" });

	return <WikiPage page={page?.wiki as Wiki} slug="home" assetURL={appInfo.assetUrl} />;
}
