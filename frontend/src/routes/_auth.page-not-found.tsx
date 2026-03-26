import { createFileRoute } from "@tanstack/react-router";
import { PageNotFound } from "../component/PageNotFound";

export const Route = createFileRoute("/_auth/page-not-found")({
	component: PageNotFound,
	head: ({ match }) => ({
		meta: [match.context.title("Page Not Found")],
	}),
});
