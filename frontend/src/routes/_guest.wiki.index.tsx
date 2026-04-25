import { useQuery } from "@connectrpc/connect-query";
import { createFileRoute } from "@tanstack/react-router";
import { WikiPage } from "../component/WikiPage.tsx";
import type { Wiki } from "../rpc/wiki/v1/wiki_pb.ts";
import { get } from "../rpc/wiki/v1/wiki-WikiService_connectquery.ts";

export const Route = createFileRoute("/_guest/wiki/")({
	component: WikiComponent,
	head: ({ match }) => {
		// TODO set title to slug
		return {
			meta: [{ name: "description", content: "Wiki" }, match.context.title("Wiki")],
		};
	},
});

function WikiComponent() {
	const { appInfo } = Route.useRouteContext();
	const { data: page, isLoading } = useQuery(get, { slug: "home" });

	if (isLoading) {
		return;
	}

	return <WikiPage page={page?.wiki as Wiki} slug="home" assetURL={appInfo.assetUrl} />;
}
