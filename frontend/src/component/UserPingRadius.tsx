import { useEffect, useMemo } from "react";
import { Circle } from "react-leaflet";
import { useMap } from "react-leaflet/hooks";
import { useMapStateCtx } from "../hooks/useMapStateCtx";

export const UserPingRadius = () => {
	const map = useMap();
	const { pos, customRange, filterByRegion } = useMapStateCtx();
	const baseOpts = { color: "green", opacity: 0.1, interactive: true };
	const markers = [
		{ ...baseOpts, radius: 3000000, color: "red" },
		{ ...baseOpts, radius: 1500000, color: "yellow" },
		{ ...baseOpts, radius: 500000, color: "green" },
	];

	useEffect(() => {
		if (pos.lat !== 0 && pos.lng !== 0) {
			map.setView(pos, 3);
		}
	}, [map, pos]);

	const c = useMemo(() => {
		return filterByRegion && <Circle center={pos} radius={customRange * 1000} color={"green"} />;
	}, [customRange, pos, filterByRegion]);

	return (
		<>
			{c}
			{pos.lat !== 0 && markers.map((m) => <Circle center={pos} key={m.radius} {...m} fillOpacity={0.1} />)}
		</>
	);
};
