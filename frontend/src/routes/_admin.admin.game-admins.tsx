import { createFileRoute } from "@tanstack/react-router";
import { AdminsEditor } from "../component/AdminsEditor.tsx";

export const Route = createFileRoute("/_admin/admin/game-admins")({
	component: AdminsEditor,
	head: () => ({
		meta: [
			{ name: "description", content: "Game Admins" },
			{ name: "keywords", content: "game, admin, management" },
			{ title: "Game Admins" },
		],
	}),
});
