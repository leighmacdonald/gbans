import { Circle } from "react-leaflet";
import type { SafeServer } from "../rpc/servers/v1/servers_pb.ts";

export const ServerMarkers = ({ servers }: { servers: SafeServer[] }) => {
	return (
		<>
			{servers.map((s) => {
				return (
					<Circle
						center={{
							lat: s.latLong?.latitude ?? 0,
							lng: s.latLong?.longitude ?? 0,
						}}
						radius={50000}
						color={"green"}
						key={`${s.nameShort}${s.serverId}`}
					/>
				);
			})}
		</>
	);
};
