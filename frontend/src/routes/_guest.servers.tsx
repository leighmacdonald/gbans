import { useQuery } from "@connectrpc/connect-query";
import Paper from "@mui/material/Paper";
import Stack from "@mui/material/Stack";
import { createFileRoute } from "@tanstack/react-router";
import type { LatLngLiteral } from "leaflet";
import { useState } from "react";
import { ServerFilters } from "../component/ServerFilters.tsx";
import { ServerList } from "../component/ServerList.tsx";
import { ServerMap } from "../component/ServerMap.tsx";
import { MapStateCtx } from "../contexts/MapStateCtx.tsx";
import type { SafeServer } from "../rpc/servers/v1/servers_pb.ts";
import { state } from "../rpc/servers/v1/servers-ServersService_connectquery.ts";
import { ensureFeatureEnabled } from "../util/features.ts";

export const Route = createFileRoute("/_guest/servers")({
	component: Servers,
	beforeLoad: ({ context }) => {
		ensureFeatureEnabled(context.appInfo.serversEnabled);
	},
	head: ({ match }) => ({
		meta: [{ name: "description", content: "Server Browser" }, match.context.title("Servers")],
	}),
});

function Servers() {
	const [pos, setPos] = useState<LatLngLiteral>({
		lat: 0.0,
		lng: 0.0,
	});
	const [customRange, setCustomRange] = useState<number>(500);
	const [selectedServers, setSelectedServers] = useState<SafeServer[]>([]);
	const [filterByRegion, setFilterByRegion] = useState<boolean>(false);
	const [showOpenOnly, setShowOpenOnly] = useState<boolean>(false);
	const [selectedRegion, setSelectedRegion] = useState<string>("any");

	const { data: servers, isLoading } = useQuery(state, {}, { refetchInterval: 5000 });

	// const { restart } = useTimer({
	// 	autoStart: true,
	// 	expiryTimestamp: new Date(),
	// 	onExpire: () => {
	// 		// TODO replace this with tan query
	// 		const ac = new AbortController();
	// 		apiGetServerStates(ac.signal)
	// 			.then((response) => {
	// 				if (!response) {
	// 					restart(nextExpiry());
	// 					return;
	// 				}
	// 				setServers(response.servers || []);
	// 				if (pos.lat === 0) {
	// 					setPos({
	// 						lat: response.lat_long.latitude,
	// 						lng: response.lat_long.longitude,
	// 					});
	// 				}
	//
	// 				restart(nextExpiry());
	// 			})
	// 			.catch(() => {
	// 				restart(nextExpiry());
	// 			});
	// 	},
	// });
	return (
		<MapStateCtx.Provider
			value={{
				servers: isLoading ? [] : (servers?.servers ?? []),
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
				<ServerList />
			</Stack>
		</MapStateCtx.Provider>
	);
}
