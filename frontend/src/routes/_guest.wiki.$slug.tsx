import { createFileRoute } from "@tanstack/react-router";
import { ErrorDetails } from "../component/ErrorDetails.tsx";
import { WikiPage } from "../component/WikiPage.tsx";
import { AppError } from "../error.tsx";
import { useSuspenseQuery } from "@connectrpc/connect-query";
import { WikiSchema } from "../rpc/wiki/v1/wiki_pb.ts";
import { get } from "../rpc/wiki/v1/wiki-WikiService_connectquery.ts";
import { create } from "@bufbuild/protobuf";
import { Privilege } from "../rpc/person/v1/privilege_pb.ts";

export const Route = createFileRoute("/_guest/wiki/$slug")({
	component: Component,
	head: ({ match, params }) => ({
		meta: [{ name: "description", content: "Wiki" }, match.context.title(params.slug)],
	}),
	errorComponent: ({ error }) => {
		if (error instanceof AppError) {
			return <ErrorDetails error={error} />;
		}
		return <div>hmmm</div>;
	},
});

function Component() {
	const { slug } = Route.useParams();
	const { appInfo } = Route.useRouteContext();

	const { data, isLoading } = useSuspenseQuery(get, { slug });

	if (isLoading) {
		return <div>loading...</div>;
	}

	const page =
		data.wiki ??
		create(WikiSchema, {
			revision: 0,
			bodyMd: "",
			slug: slug,
			permissionLevel: Privilege.GUEST,
		});

	return <WikiPage slug={slug} page={page} assetURL={appInfo.assetUrl} />;
}
