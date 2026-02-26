import { createFileRoute } from "@tanstack/react-router";
import { AdminsEditor } from "../component/AdminsEditor.tsx";

export const Route = createFileRoute("/_admin/admin/game-admins")({
	component: AdminsEditor,
	loader: ({ context }) => ({
		appInfo: context.appInfo,
	}),
	head: ({ loaderData }) => ({
		meta: [
			{ name: "description", content: "Game Admins" },
			{ title: `Game Admins - ${loaderData?.appInfo.site_name}` },
		],
	}),
});
