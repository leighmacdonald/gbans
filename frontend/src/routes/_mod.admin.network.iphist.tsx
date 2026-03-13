import Grid from "@mui/material/Grid";
import { createFileRoute } from "@tanstack/react-router";

import { IPHistoryTable } from "../component/table/IPHistoryTable.tsx";

export const Route = createFileRoute("/_mod/admin/network/iphist")({
	component: AdminNetworkPlayerIPHistory,
	head: ({ match }) => ({
		meta: [{ name: "description", content: "Player IP History" }, match.context.title("Player IP History")],
	}),
});

function AdminNetworkPlayerIPHistory() {
	return (
		<Grid container spacing={2}>
			<Grid size={{ xs: 12 }}>
				<IPHistoryTable />
			</Grid>
		</Grid>
	);
}
