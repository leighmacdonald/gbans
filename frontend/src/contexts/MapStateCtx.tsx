import type { LatLngLiteral } from "leaflet";
import { createContext } from "react";
import type { SafeServer } from "../rpc/servers/v1/servers_pb.ts";
import { noop } from "../util/lists";

export type MapState = {
	pos: LatLngLiteral;
	setPos: (pos: LatLngLiteral) => void;

	customRange: number;
	setCustomRange: (radius: number) => void;

	servers: SafeServer[];

	selectedServers: SafeServer[];
	setSelectedServers: (servers: SafeServer[]) => void;

	filterByRegion: boolean;
	setFilterByRegion: (enable: boolean) => void;

	showOpenOnly: boolean;
	setShowOpenOnly: (enabled: boolean) => void;

	selectedRegion: string;
	setSelectedRegion: (regions: string) => void;
};

export const MapStateCtx = createContext<MapState>({
	pos: { lat: 0.0, lng: 0.0 },
	setPos: noop,
	customRange: 1500,
	setCustomRange: noop,
	servers: [],
	selectedServers: [],
	setSelectedServers: noop,
	filterByRegion: true,
	setFilterByRegion: noop,
	showOpenOnly: false,
	setShowOpenOnly: noop,
	selectedRegion: "any",
	setSelectedRegion: noop,
});
