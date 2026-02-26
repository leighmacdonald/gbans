import { createFileRoute } from "@tanstack/react-router";

export const Route = createFileRoute("/_mod/admin/news")({
	component: RouteComponent,
});

function RouteComponent() {
	return <div>Hello "/_mod/admin/news"!</div>;
}
