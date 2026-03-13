import Paper from "@mui/material/Paper";
import Stack from "@mui/material/Stack";
import { createFileRoute } from "@tanstack/react-router";
import type { LatLngLiteral } from "leaflet";
import { useState } from "react";
import { useTimer } from "react-timer-hook";
import { apiGetServerStates } from "../api";
import { QueueHelp } from "../component/queue/QueueHelp.tsx";
import { ServerFilters } from "../component/ServerFilters.tsx";
import { ServerList } from "../component/ServerList.tsx";
import { ServerMap } from "../component/ServerMap.tsx";
import { MapStateCtx } from "../contexts/MapStateCtx.tsx";
import { useAuth } from "../hooks/useAuth.ts";
import { PermissionLevel } from "../schema/people.ts";
import type { BaseServer } from "../schema/server.ts";
import { ensureFeatureEnabled } from "../util/features.ts";

export const Route = createFileRoute("/_guest/servers")({
	component: Servers,
	beforeLoad: ({ context }) => {
		ensureFeatureEnabled(context.appInfo.servers_enabled);
	},
	head: ({ match }) => ({
		meta: [{ name: "description", content: "Server Browser" }, match.context.title("Servers")],
	}),
});

function Servers() {
	const [servers, setServers] = useState<BaseServer[]>([]);
	const { hasPermission } = useAuth();
	const [pos, setPos] = useState<LatLngLiteral>({
		lat: 0.0,
		lng: 0.0,
	});
	const [customRange, setCustomRange] = useState<number>(500);
	const [selectedServers, setSelectedServers] = useState<BaseServer[]>([]);
	const [filterByRegion, setFilterByRegion] = useState<boolean>(false);
	const [showOpenOnly, setShowOpenOnly] = useState<boolean>(false);
	const [selectedRegion, setSelectedRegion] = useState<string>("any");
	const [showHelp] = useState<boolean>(false);

	const interval = 5;

	const nextExpiry = () => {
		const t0 = new Date();
		t0.setSeconds(t0.getSeconds() + interval);
		return t0;
	};

	const { restart } = useTimer({
		autoStart: true,
		expiryTimestamp: new Date(),
		onExpire: () => {
			apiGetServerStates()
				.then((response) => {
					if (!response) {
						restart(nextExpiry());
						return;
					}
					setServers(response.servers || []);
					if (pos.lat === 0) {
						setPos({
							lat: response.lat_long.latitude,
							lng: response.lat_long.longitude,
						});
					}

					restart(nextExpiry());
				})
				.catch(() => {
					restart(nextExpiry());
				});
		},
	});
	return (
		<MapStateCtx.Provider
			value={{
				servers,
				setServers,
				customRange,
				setCustomRange,
				pos,
				setPos,
				selectedServers,
				setSelectedServers,
				filterByRegion,
				setFilterByRegion,
				showOpenOnly,
				setShowOpenOnly,
				selectedRegion,
				setSelectedRegion,
			}}
		>
			<Stack spacing={3}>
				<Paper elevation={3}>
					<ServerMap />
				</Paper>
				<ServerFilters />
				{hasPermission(PermissionLevel.Moderator) && showHelp && <QueueHelp />}
				<ServerList />
			</Stack>
		</MapStateCtx.Provider>
	);
}
