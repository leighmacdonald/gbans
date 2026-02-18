import type { LatLngLiteral } from "leaflet";
import { createContext } from "react";
import type { BaseServer } from "../schema/server.ts";
import { noop } from "../util/lists";

export type MapState = {
	pos: LatLngLiteral;
	setPos: (pos: LatLngLiteral) => void;

	customRange: number;
	setCustomRange: (radius: number) => void;

	servers: BaseServer[];
	setServers: (servers: BaseServer[]) => void;

	selectedServers: BaseServer[];
	setSelectedServers: (servers: BaseServer[]) => void;

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
	setServers: noop,
	selectedServers: [],
	setSelectedServers: noop,
	filterByRegion: true,
	setFilterByRegion: noop,
	showOpenOnly: false,
	setShowOpenOnly: noop,
	selectedRegion: "any",
	setSelectedRegion: noop,
});
