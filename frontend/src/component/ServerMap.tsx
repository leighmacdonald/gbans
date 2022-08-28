import React, { useEffect, useMemo } from 'react';
import { Circle, MapContainer, Marker, TileLayer, useMap } from 'react-leaflet';
import 'leaflet/dist/leaflet.css';
import * as markerIcon2x from 'leaflet/dist/images/marker-icon-2x.png';
import * as markerIcon from 'leaflet/dist/images/marker-icon.png';
import * as markerShadow from 'leaflet/dist/images/marker-shadow.png';
import L from 'leaflet';
import { useMapStateCtx } from '../contexts/MapStateCtx';

// Workaround for leaflet not loading icons properly in react
// eslint-disable-next-line @typescript-eslint/ban-ts-comment
// @ts-ignore
delete L.Icon.Default.prototype._getIconUrl;
L.Icon.Default.mergeOptions({
    iconRetinaUrl: markerIcon2x.default,
    iconUrl: markerIcon.default,
    shadowUrl: markerShadow.default
});

const UserPosition = () => {
    const map = useMap();
    const { setPos } = useMapStateCtx();

    useEffect(() => {
        const defPos = { lat: 42.434719, lng: -83.985001 };
        if ('geolocation' in navigator) {
            navigator.geolocation.getCurrentPosition(
                (pos) => {
                    const userPos = {
                        lat: pos.coords.latitude,
                        lng: pos.coords.longitude
                    };
                    map.setView(userPos, 3);
                    setPos(userPos);
                },
                () => {
                    map.setView(defPos, 4);
                    setPos(defPos);
                }
            );
        } else {
            map.setView(defPos);
            setPos(defPos);
        }
    }, [map, setPos]);

    return null;
};

export const ServerMarkers = () => {
    const { servers } = useMapStateCtx();
    const d = useMemo(
        () =>
            (servers || []).map((s) => {
                //const dis = getDistance(pos, { lat: s.latitude, lng: s.longitude }) / 1000;
                return (
                    <Circle
                        center={{
                            lat: s.latitude,
                            lng: s.longitude
                        }}
                        radius={50000}
                        color={'green'}
                        key={s.name_short}
                    />
                );
            }),
        [servers]
    );
    return <>{servers && d}</>;
};

const UserPositionMarker = () => {
    const { pos } = useMapStateCtx();
    return (
        <>
            {pos.lat != 0 && (
                <Marker autoPan={true} title={'You'} position={pos} />
            )}
        </>
    );
};

export const UserPingRadius = () => {
    const { pos, customRange, filterByRegion } = useMapStateCtx();
    const baseOpts = { color: 'green', opacity: 0.1, interactive: true };
    const markers = [
        { ...baseOpts, radius: 3000000, color: 'red' },
        { ...baseOpts, radius: 1500000, color: 'yellow' },
        { ...baseOpts, radius: 500000, color: 'green' }
    ];
    const c = useMemo(() => {
        return (
            filterByRegion && (
                <Circle
                    center={pos}
                    radius={customRange * 1000}
                    color={'green'}
                />
            )
        );
    }, [customRange, pos, filterByRegion]);

    return (
        <>
            {c}
            {pos.lat != 0 &&
                markers.map((m) => (
                    <Circle
                        center={pos}
                        key={m.radius}
                        {...m}
                        fillOpacity={0.1}
                    />
                ))}
        </>
    );
};

export const ServerMap = () => {
    return (
        <MapContainer
            zoom={3}
            scrollWheelZoom={true}
            id={'map'}
            style={{ height: '500px', width: '100%' }}
            attributionControl={true}
            minZoom={3}
            worldCopyJump={true}
        >
            <TileLayer
                url="https://{s}.tile.openstreetmap.org/{z}/{x}/{y}.png"
                attribution={'© OpenStreetMap contributors '}
            />

            <UserPosition />
            <UserPingRadius />
            <ServerMarkers />
            <UserPositionMarker />
        </MapContainer>
    );
};
