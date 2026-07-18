import { ConnectError } from "@connectrpc/connect";
import { useQuery } from "@connectrpc/connect-query";
import { createFileRoute } from "@tanstack/react-router";
import { useEffect } from "react";
import { ErrorDetails } from "../component/ErrorDetails.tsx";
import {  LoadingPlaceholder} from "../component/LoadingPlaceholder.tsx"
import { WikiPage } from "../component/WikiPage.tsx";
import { AppError } from "../error.tsx";
import { get } from "../rpc/wiki/v1/wiki-WikiService_connectquery.ts";

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
	const { data, isLoading, isError, error } = useQuery(get, { slug }, { retry: false });

	if (isError) {
		return <ErrorDetails error={error} />;
	}

	if (isLoading || !data?.wiki) {
		return <LoadingPlaceholder />;
	}

	return <WikiPage slug={slug} page={data?.wiki} assetURL={appInfo.assetUrl} />;
}
