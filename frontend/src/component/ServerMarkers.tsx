import { Circle, Popup } from "react-leaflet";
import type { SafeServer } from "../rpc/servers/v1/servers_pb.ts";

function ServerRow({ server }: { server: SafeServer }) {
	const bots = server.bot > 0 ? ` (+${server.bot})` : "";
	return (
		<div>
			<b>{server.name}</b> · {server.map || "—"} · {server.humans}/{server.maxPlayers}
			{bots}
		</div>
	);
}

export const ServerMarkers = ({ servers }: { servers: SafeServer[] }) => {
	// Several servers usually share one datacenter location; group them so a
	// single circle's popup lists every server there instead of only the top one.
	const groups = new Map<string, SafeServer[]>();
	for (const server of servers) {
		const key = `${server.latLong?.latitude ?? 0},${server.latLong?.longitude ?? 0}`;
		const group = groups.get(key);
		if (group) {
			group.push(server);
		} else {
			groups.set(key, [server]);
		}
	}

	return (
		<>
			{[...groups.entries()].map(([key, group]) => {
				const [first] = group;
				return (
					<Circle
						center={{
							lat: first.latLong?.latitude ?? 0,
							lng: first.latLong?.longitude ?? 0,
						}}
						radius={50000}
						color={"green"}
						key={key}
						eventHandlers={{
							mouseover: (event) => event.target.openPopup(),
							mouseout: (event) => event.target.closePopup(),
						}}
					>
						<Popup closeButton={false} minWidth={200} maxWidth={280} maxHeight={300}>
							<div style={{ fontSize: "0.8rem", lineHeight: 1.5 }}>
								<div>
									<b>
										{group.length > 1 ? `${group.length} servers · ` : ""}
										{first.region || "—"} · {Math.round(first.distance)} km
									</b>
								</div>
								{group.map((server) => (
									<ServerRow key={`${server.nameShort}${server.serverId}`} server={server} />
								))}
							</div>
						</Popup>
					</Circle>
				);
			})}
		</>
	);
};
