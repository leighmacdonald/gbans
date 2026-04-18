import { createFileRoute } from "@tanstack/react-router";
import { ErrorDetails } from "../component/ErrorDetails.tsx";
import { WikiPage } from "../component/WikiPage.tsx";
import { AppError } from "../error.tsx";
import { PermissionLevel } from "../schema/people.ts";
import { useSuspenseQuery} from "@connectrpc/connect-query";
import {get} from "../rpc/wiki/v1/wiki-WikiService_connectquery.ts";

export const Route = createFileRoute("/_guest/wiki/$slug")({
	component: Wiki,
	head: ({ match, params }) => ({
		meta: [{ name: "description", content: "Wiki" }, match.context.title(params.slug)],
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
	const { appInfo } = Route.useRouteContext();

    const {data, isLoading} = useSuspenseQuery(get, {slug});

    if (isLoading) {
        return <div>loading...</div>;
    }


    const page = data.wiki ?? {
            revision: 0,
            body_md: "",
            slug: slug,
            permission_level: PermissionLevel.Guest,
            created_on: new Date(),
            updated_on: new Date(),
        }

	return <WikiPage slug={slug} page={page} assetURL={appInfo.assetUrl} />;
}
