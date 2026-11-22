import { useQuery } from "@connectrpc/connect-query";
import MonitorHeartIcon from "@mui/icons-material/MonitorHeart";
import RefreshIcon from "@mui/icons-material/Refresh";
import Alert from "@mui/material/Alert";
import Box from "@mui/material/Box";
import Button from "@mui/material/Button";
import Card from "@mui/material/Card";
import CardContent from "@mui/material/CardContent";
import Chip from "@mui/material/Chip";
import Grid from "@mui/material/Grid";
import Stack from "@mui/material/Stack";
import Table from "@mui/material/Table";
import TableBody from "@mui/material/TableBody";
import TableCell from "@mui/material/TableCell";
import TableHead from "@mui/material/TableHead";
import TableRow from "@mui/material/TableRow";
import Typography from "@mui/material/Typography";
import { createFileRoute } from "@tanstack/react-router";
import { ContainerWithHeaderAndButtons } from "../component/ContainerWithHeaderAndButtons.tsx";
import { LoadingPlaceholder } from "../component/LoadingPlaceholder.tsx";
import { type HealthItem, HealthStatus } from "../rpc/health/v1/health_pb.ts";
import { health } from "../rpc/health/v1/health-HealthService_connectquery.ts";
import { renderTimestamp } from "../util/time.ts";

export const Route = createFileRoute("/_admin/admin/health")({
	head: ({ match }) => ({
		meta: match.context.meta({ title: "System Health", description: "Service and server health overview." }),
	}),
	component: AdminHealth,
});

const statusColour: Record<HealthStatus, "success" | "warning" | "error" | "default"> = {
	[HealthStatus.UNSPECIFIED]: "default",
	[HealthStatus.OK]: "success",
	[HealthStatus.WARN]: "warning",
	[HealthStatus.ERROR]: "error",
	[HealthStatus.DISABLED]: "default",
};

const statusLabel: Record<HealthStatus, string> = {
	[HealthStatus.UNSPECIFIED]: "Unknown",
	[HealthStatus.OK]: "OK",
	[HealthStatus.WARN]: "Warning",
	[HealthStatus.ERROR]: "Error",
	[HealthStatus.DISABLED]: "Disabled",
};

function HealthCard({ item }: { item: HealthItem }) {
	return (
		<Card variant={"outlined"} sx={{ height: "100%" }}>
			<CardContent>
				<Stack direction={"row"} justifyContent={"space-between"} alignItems={"center"} spacing={1}>
					<Typography variant={"subtitle2"}>{item.name}</Typography>
					<Chip size={"small"} label={statusLabel[item.status]} color={statusColour[item.status]} />
				</Stack>
				<Typography variant={"body2"} sx={{ mt: 1 }}>
					{item.detail}
				</Typography>
				{item.lastCheck && (
					<Typography variant={"caption"} color={"text.secondary"}>
						Checked {renderTimestamp(item.lastCheck)}
					</Typography>
				)}
			</CardContent>
		</Card>
	);
}

function AdminHealth() {
	const { data, isLoading, isError, refetch, isFetching } = useQuery(health, {}, { refetchInterval: 15_000 });

	return (
		<Grid container spacing={2}>
			<Grid size={{ xs: 12 }}>
				<ContainerWithHeaderAndButtons
					title={"System Health"}
					iconLeft={<MonitorHeartIcon />}
					buttons={[
						<Button
							key={"refresh"}
							startIcon={<RefreshIcon />}
							size={"small"}
							disabled={isFetching}
							onClick={() => {
								void refetch();
							}}
						>
							Refresh
						</Button>,
					]}
				>
					{isLoading && <LoadingPlaceholder />}
					{isError && <Alert severity={"error"}>Failed to load health information.</Alert>}
					{data && (
						<>
							<Grid container spacing={2}>
								{data.items.map((item) => (
									<Grid key={item.name} size={{ xs: 12, sm: 6, md: 4 }}>
										<HealthCard item={item} />
									</Grid>
								))}
							</Grid>

							<Box sx={{ mt: 3 }}>
								<Typography variant={"h6"} sx={{ mb: 1 }}>
									Game Servers
								</Typography>
								<Table size={"small"}>
									<TableHead>
										<TableRow>
											<TableCell>Server</TableCell>
											<TableCell>Enabled</TableCell>
											<TableCell>Last RCON Success</TableCell>
											<TableCell>Last Status Update</TableCell>
											<TableCell>Last Auth</TableCell>
											<TableCell>STV Port</TableCell>
										</TableRow>
									</TableHead>
									<TableBody>
										{data.servers.map((server) => (
											<TableRow key={server.serverId}>
												<TableCell>{server.name}</TableCell>
												<TableCell>{server.enabled ? "Yes" : "No"}</TableCell>
												<TableCell>{renderTimestamp(server.lastRconSuccess)}</TableCell>
												<TableCell>{renderTimestamp(server.lastStatusUpdate)}</TableCell>
												<TableCell>{renderTimestamp(server.lastAuth)}</TableCell>
												<TableCell>{server.stvPort > 0 ? server.stvPort : "-"}</TableCell>
											</TableRow>
										))}
									</TableBody>
								</Table>
							</Box>
						</>
					)}
				</ContainerWithHeaderAndButtons>
			</Grid>
		</Grid>
	);
}
