import { useContext } from "react";
import { MapStateCtx } from "../contexts/MapStateCtx.tsx";

export const useMapStateCtx = () => useContext(MapStateCtx);
