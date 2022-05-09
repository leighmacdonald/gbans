import { createContext, useContext } from 'react';
import noop from 'lodash-es/noop';
import { LatLngLiteral } from 'leaflet';
import { ServerState } from '../api';

export type MapState = {
    pos: LatLngLiteral;
    setPos: (pos: LatLngLiteral) => void;

    servers: ServerState[];
    setServers: (servers: ServerState[]) => void;
};

export const MapStateCtx = createContext<MapState>({
    pos: { lat: 0.0, lng: 0.0 },
    setPos: noop,
    servers: [],
    setServers: noop
});

export const useMapStateCtx = () => useContext(MapStateCtx);
