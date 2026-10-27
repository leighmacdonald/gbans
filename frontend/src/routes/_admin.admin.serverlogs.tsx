import { useSuspenseQuery } from "@connectrpc/connect-query";
import Grid from "@mui/material/Grid";
import Typography from "@mui/material/Typography";
import { createFileRoute, stripSearchParams } from "@tanstack/react-router";
import { z } from "zod/v4";
import { LogTerminal } from "../component/LogTerminal.tsx";
import { servers } from "../rpc/servers/v1/servers-ServersService_connectquery.ts";

const defaultValues = { serverId: undefined as number | undefined };

const validateSearch = z.object({
	serverId: z.coerce.number().int().positive().optional(),
});

export const Route = createFileRoute("/_admin/admin/serverlogs")({
	validateSearch,
	search: {
		middlewares: [stripSearchParams(defaultValues)],
	},
	head: ({ match }) => {
		return {
			meta: match.context.meta({ title: "Server Logs", description: "Browse live server logs." }),
		};
	},
	component: AdminServerLogs,
});

function AdminServerLogs() {
	const search = Route.useSearch();
	const navigate = Route.useNavigate();

	const { data: serverData } = useSuspenseQuery(servers);

	const serverId = search.serverId ?? serverData.servers[0]?.serverId;

	return (
		<Grid container spacing={2}>
			<Grid size={{ xs: 12 }}>
				{serverId ? (
					<LogTerminal
						servers={serverData.servers}
						serverId={serverId}
						onServerChange={(id) => navigate({ to: Route.fullPath, search: { serverId: id } })}
					/>
				) : (
					<Typography>No servers available.</Typography>
				)}
			</Grid>
		</Grid>
	);
}
