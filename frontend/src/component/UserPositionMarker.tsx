import { Marker } from "react-leaflet";
import { useMapStateCtx } from "../hooks/useMapStateCtx.ts";

export const UserPositionMarker = () => {
	const { pos } = useMapStateCtx();
	return <>{pos.lat !== 0 && <Marker autoPan={true} title={"You"} position={pos} />}</>;
};
