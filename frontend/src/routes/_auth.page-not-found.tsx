import { createFileRoute } from "@tanstack/react-router";
import { PageNotFound } from "../component/PageNotFound";

export const Route = createFileRoute("/_auth/page-not-found")({
	component: PageNotFound,
	head: ({ match }) => ({
		meta: match.context.meta({
			title: "Page Not Found",
			description: "The page you are looking for does not exist.",
		}),
	}),
});
