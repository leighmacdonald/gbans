import React, { useEffect, useMemo } from 'react';
import { Circle, MapContainer, Marker, TileLayer, useMap } from 'react-leaflet';
import { useMapStateCtx } from '../contexts/MapStateCtx';
import 'leaflet/dist/leaflet.css';

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
                    map.setView(userPos);
                    setPos(userPos);
                },
                () => {
                    map.setView(defPos);
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
            servers.map((s) => {
                //const dis = getDistance(pos, { lat: s.latitude, lng: s.longitude }) / 1000;
                return (
                    <Circle
                        center={[s.latitude, s.longitude]}
                        radius={50000}
                        color={'green'}
                        key={s.server_id}
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

export const ServerMap = () => {
    return (
        <MapContainer
            zoom={4}
            scrollWheelZoom={true}
            id={'map'}
            style={{ height: '500px', width: '100%' }}
            attributionControl={true}
            minZoom={3}
            worldCopyJump={true}
        >
            <TileLayer url="https://{s}.tile.openstreetmap.org/{z}/{x}/{y}.png" />
            <UserPosition />
            <ServerMarkers />
            <UserPositionMarker />
        </MapContainer>
    );
};
