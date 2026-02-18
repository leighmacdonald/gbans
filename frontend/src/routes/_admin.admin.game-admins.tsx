import { createFileRoute } from "@tanstack/react-router";
import { AdminsEditor } from "../component/AdminsEditor.tsx";

export const Route = createFileRoute("/_admin/admin/game-admins")({
	component: AdminsEditor,
});
