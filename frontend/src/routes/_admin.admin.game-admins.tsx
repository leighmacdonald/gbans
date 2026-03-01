import { createFileRoute } from "@tanstack/react-router";
import { AdminsEditor } from "../component/AdminsEditor.tsx";

export const Route = createFileRoute("/_admin/admin/game-admins")({
	component: AdminsEditor,
	head: ({ match }) => {
		return {
			meta: [{ name: "description", content: "Game Admins" }, match.context.title("Game Admins")],
		};
	},
});
