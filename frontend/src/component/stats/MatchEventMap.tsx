import "leaflet/dist/leaflet.css";
import MapIcon from "@mui/icons-material/Map";
import Alert from "@mui/material/Alert";
import Box from "@mui/material/Box";
import FormControlLabel from "@mui/material/FormControlLabel";
import MenuItem from "@mui/material/MenuItem";
import Paper from "@mui/material/Paper";
import Select from "@mui/material/Select";
import Slider from "@mui/material/Slider";
import Stack from "@mui/material/Stack";
import Switch from "@mui/material/Switch";
import Typography from "@mui/material/Typography";
import { CRS } from "leaflet";
import { Fragment, useEffect, useMemo, useState } from "react";
import { CircleMarker, ImageOverlay, MapContainer, Polyline, Popup, Tooltip } from "react-leaflet";
import { useMap } from "react-leaflet/hooks";
import { Team } from "../../rpc/stats/v1/stats_pb.ts";
import { formatMatchClock } from "../../util/time.ts";
import { ContainerWithHeaderAndButtons } from "../ContainerWithHeaderAndButtons.tsx";
import { humanizeEventType } from "./EventFeedTable.tsx";
import {
	eventMarkerColor,
	eventTypeColor,
	fetchReferenceFile,
	fetchWorldFile,
	overviewPixelBounds,
	overviewUrls,
	paddedOverviewPixelBounds,
	pixelBoundsToLatLng,
	type ReferencePoint,
	referencePointLabel,
	referencePointStyle,
	referenceUrls,
	type WorldFile,
	worldToPixel,
} from "./killMap.ts";
import { type EventFeedEntry, eventMapLayerKey, eventMapLayerLabel, type MatchView } from "./match";

const MAP_HEIGHT = 480;

const clamp = (v: number, lo: number, hi: number): number => Math.min(hi, Math.max(lo, v));

const projectPoint = (worldFile: WorldFile, imageHeight: number, x: number, y: number): [number, number] => {
	const { px, py } = worldToPixel(worldFile, x, y);
	return [imageHeight - py, px];
};

const FitOverview = ({ bounds }: { bounds: [[number, number], [number, number]] }) => {
	const map = useMap();
	useEffect(() => {
		map.invalidateSize();
		map.fitBounds(bounds);
	}, [map, bounds]);
	return null;
};

const hasPosition = (event: EventFeedEntry): boolean =>
	event.eventX !== null && event.eventY !== null && Number.isFinite(event.eventX) && Number.isFinite(event.eventY);

/**
 * True when a kill has distinct attacker and victim positions, so it draws
 * as an attacker→victim line instead of a point marker. World kills (no
 * attacker) and kills missing an end fall back to the victim point.
 */
const hasKillLine = (event: EventFeedEntry): boolean =>
	event.eventType === "kill" &&
	event.attackerX !== null &&
	event.attackerY !== null &&
	event.eventX !== null &&
	event.eventY !== null &&
	Number.isFinite(event.attackerX) &&
	Number.isFinite(event.attackerY) &&
	(event.attackerX !== event.eventX || event.attackerY !== event.eventY);

const teamLabel = (team: Team): string => {
	if (team === Team.BLU) {
		return "BLU";
	}
	if (team === Team.RED) {
		return "RED";
	}
	return "";
};

export const MatchEventMap = ({ summary, mapName }: { summary: MatchView; mapName: string }) => {
	const [worldFile, setWorldFile] = useState<WorldFile | null>(null);
	const [imgSize, setImgSize] = useState<{ width: number; height: number } | null>(null);
	const [referencePoints, setReferencePoints] = useState<ReferencePoint[]>([]);
	const [loadError, setLoadError] = useState<string | null>(null);
	const [roundFilter, setRoundFilter] = useState<number>(0);
	const [showReferencePoints, setShowReferencePoints] = useState(true);
	const [eventVisibility, setEventVisibility] = useState<Record<string, boolean>>({});
	/** Scrub position in demo ticks; null means "show everything". */
	const [tickFilter, setTickFilter] = useState<number | null>(null);

	/**
	 * Only positional events with a layer key get map geometries; the rest
	 * (non-positional events, mini/gunslinger builds, unknown building kinds)
	 * stay in the feed table.
	 */
	const mappableEvents = useMemo(
		() => summary.events.filter((event) => hasPosition(event) && eventMapLayerKey(event) !== null),
		[summary.events],
	);

	const layerKeys = useMemo(() => {
		const seen = new Set<string>();
		for (const event of mappableEvents) {
			const key = eventMapLayerKey(event);
			if (key !== null) {
				seen.add(key);
			}
		}
		return [...seen].toSorted();
	}, [mappableEvents]);

	// Default new layers to visible without clobbering user toggles.
	useEffect(() => {
		setEventVisibility((prev) => {
			let changed = false;
			const next = { ...prev };
			for (const key of layerKeys) {
				if (!(key in next)) {
					next[key] = true;
					changed = true;
				}
			}
			return changed ? next : prev;
		});
	}, [layerKeys]);

	useEffect(() => {
		let cancelled = false;
		setWorldFile(null);
		setImgSize(null);
		setReferencePoints([]);
		setLoadError(null);
		if (mappableEvents.length === 0) {
			return;
		}
		const urls = overviewUrls(mapName);
		fetchReferenceFile(referenceUrls(mapName)).then((points) => {
			if (!cancelled && points) {
				setReferencePoints(points);
			}
		});
		fetchWorldFile(urls.worldFile).then((parsed) => {
			if (cancelled) {
				return;
			}
			if (!parsed) {
				setLoadError(`No overview map available for ${mapName}.`);
				return;
			}
			const img = new Image();
			img.onload = () => {
				if (!cancelled) {
					setImgSize({ width: img.naturalWidth, height: img.naturalHeight });
					setWorldFile(parsed);
				}
			};
			img.onerror = () => {
				if (!cancelled) {
					setLoadError(`No overview map available for ${mapName}.`);
				}
			};
			img.src = urls.image;
		});
		return () => {
			cancelled = true;
		};
	}, [mapName, mappableEvents.length]);

	const nameOf = (steamId: string): string => {
		if (steamId === "" || steamId === "0") {
			return "World";
		}
		return summary.players[steamId]?.name ?? steamId;
	};

	const rounds = useMemo(() => {
		const seen = new Set<number>();
		for (const event of mappableEvents) {
			if (event.round > 0) {
				seen.add(event.round);
			}
		}
		return [...seen].toSorted((a, b) => a - b);
	}, [mappableEvents]);

	const roundLabel = (round: number): string => {
		const r = summary.rounds.find((x) => x.round === round);
		return r ? `Round ${round} (${r.scoreBlu} – ${r.scoreRed})` : `Round ${round}`;
	};

	/** Demo-tick range of the mappable events, bounding the scrub slider. */
	const minTick = useMemo(
		() => mappableEvents.reduce((min, e) => Math.min(min, e.tick), Number.MAX_SAFE_INTEGER),
		[mappableEvents],
	);
	const maxTick = useMemo(() => mappableEvents.reduce((max, e) => Math.max(max, e.tick), 0), [mappableEvents]);

	/** Effective scrub position; defaults to the latest tick (show everything). */
	const scrubTick = tickFilter ?? maxTick;

	// Reset the scrub position when switching matches.
	useEffect(() => {
		setTickFilter(null);
	}, [mapName]);

	const filteredEvents = useMemo(() => {
		const roundFiltered =
			roundFilter === 0 ? mappableEvents : mappableEvents.filter((e) => e.round === roundFilter);
		// Tick scrub: only draw events up to the scrub position, oldest first.
		return roundFiltered
			.filter((e) => e.tick <= scrubTick)
			.toSorted((a, b) => a.tick - b.tick || a.matchEventId.localeCompare(b.matchEventId));
	}, [mappableEvents, roundFilter, scrubTick]);

	const visibleEvents = useMemo(
		() => filteredEvents.filter((event) => eventVisibility[eventMapLayerKey(event) as string] !== false),
		[filteredEvents, eventVisibility],
	);

	const countsByLayer = useMemo(() => {
		const counts = new Map<string, number>();
		for (const event of filteredEvents) {
			const key = eventMapLayerKey(event);
			if (key !== null) {
				counts.set(key, (counts.get(key) ?? 0) + 1);
			}
		}
		return counts;
	}, [filteredEvents]);

	const referenceMarkers = useMemo(() => {
		if (!worldFile || !imgSize) {
			return [];
		}
		return referencePoints.map((point, index) => ({
			key: `${point.classname}-${point.targetname}-${point.team}-${point.x}-${point.y}-${index}`,
			center: projectPoint(worldFile, imgSize.height, point.x, point.y),
			label: referencePointLabel(point),
			style: referencePointStyle(point),
		}));
	}, [worldFile, imgSize, referencePoints]);

	const overlayBounds = useMemo(() => (imgSize ? overviewPixelBounds(imgSize) : null), [imgSize]);
	const viewLatLngBounds = useMemo(
		() => (imgSize ? pixelBoundsToLatLng(imgSize.height, paddedOverviewPixelBounds(imgSize)) : null),
		[imgSize],
	);
	const overlayLatLngBounds = useMemo(
		() => (imgSize && overlayBounds ? pixelBoundsToLatLng(imgSize.height, overlayBounds) : null),
		[imgSize, overlayBounds],
	);

	if (mappableEvents.length === 0) {
		return (
			<Paper sx={{ padding: 2 }}>
				<Alert severity="info">No positioned events were recorded for this match.</Alert>
			</Paper>
		);
	}

	if (loadError) {
		return (
			<Paper sx={{ padding: 2 }}>
				<Alert severity="info">{loadError}</Alert>
			</Paper>
		);
	}

	if (!worldFile || !imgSize || !viewLatLngBounds || !overlayLatLngBounds) {
		return (
			<Paper sx={{ padding: 2 }}>
				<Typography variant="body2" color="textSecondary">
					Loading overview…
				</Typography>
			</Paper>
		);
	}

	const toLatLng = (x: number, y: number): [number, number] => {
		const [lat, lng] = projectPoint(worldFile, imgSize.height, x, y);
		return [clamp(lat, 0, imgSize.height), clamp(lng, 0, imgSize.width)];
	};

	const markerColor = (event: EventFeedEntry): string =>
		eventMarkerColor(eventMapLayerKey(event) ?? event.eventType, event.team);

	const roundFilterControl = (
		<Stack key="event-map-round-filter" direction="row" spacing={1} sx={{ alignItems: "center" }}>
			<Typography variant="body2" sx={{ color: "common.white" }}>
				Round
			</Typography>
			<Select
				size="small"
				value={roundFilter}
				onChange={(e) => setRoundFilter(Number(e.target.value))}
				sx={{
					minWidth: 160,
					color: "common.white",
					"& .MuiOutlinedInput-notchedOutline": { borderColor: "rgba(255, 255, 255, 0.5)" },
					"&:hover .MuiOutlinedInput-notchedOutline": { borderColor: "common.white" },
					"& .MuiSvgIcon-root": { color: "common.white" },
				}}
			>
				<MenuItem value={0}>All rounds</MenuItem>
				{rounds.map((r) => (
					<MenuItem key={r} value={r}>
						{roundLabel(r)}
					</MenuItem>
				))}
			</Select>
		</Stack>
	);

	return (
		<ContainerWithHeaderAndButtons
			title="Event Map"
			iconLeft={<MapIcon />}
			buttons={[roundFilterControl]}
			padding={0}
			spacing={0}
		>
			<Box sx={{ paddingX: 1.5, paddingTop: 1.5 }}>
				<MapContainer
					crs={CRS.Simple}
					attributionControl={false}
					center={[imgSize.height / 2, imgSize.width / 2]}
					zoom={0}
					minZoom={-5}
					maxZoom={3}
					zoomSnap={0.25}
					zoomDelta={0.25}
					maxBounds={viewLatLngBounds}
					maxBoundsViscosity={1}
					scrollWheelZoom={false}
					style={{ height: MAP_HEIGHT, width: "100%", background: "transparent" }}
				>
					<FitOverview bounds={viewLatLngBounds} />
					<ImageOverlay url={overviewUrls(mapName).image} bounds={overlayLatLngBounds} />
					{showReferencePoints &&
						referenceMarkers.map((marker) => (
							<CircleMarker
								key={marker.key}
								center={marker.center}
								radius={marker.style.radius}
								pathOptions={{
									color: marker.style.color,
									fillColor: marker.style.fillColor,
									fillOpacity: marker.style.fillOpacity,
									weight: marker.style.weight,
								}}
							>
								<Tooltip>{marker.label}</Tooltip>
							</CircleMarker>
						))}
					{visibleEvents.map((event) => {
						const color = markerColor(event);
						const team = teamLabel(event.team);
						const popup = (
							<Popup>
								<Stack spacing={0.5}>
									<Typography variant="body2" sx={{ fontWeight: 700 }}>
										{event.summary}
									</Typography>
									<Typography variant="caption">
										{humanizeEventType(event.eventType)}
										{event.building ? ` · ${event.building}` : ""}
										{event.weapon ? ` · ${event.weapon}` : ""}
										{team ? ` · ${team}` : ""}
									</Typography>
									<Typography variant="caption">
										Round {event.round > 0 ? event.round : "—"} · Tick {event.tick} ·{" "}
										{formatMatchClock(event.ticksSinceStart)}
									</Typography>
									<Typography variant="caption" color="textSecondary">
										{nameOf(event.actorSteamId)}
										{event.targetSteamId &&
										event.targetSteamId !== "" &&
										event.targetSteamId !== "0"
											? ` → ${nameOf(event.targetSteamId)}`
											: ""}
									</Typography>
								</Stack>
							</Popup>
						);
						// Kills with both ends draw an attacker→victim line plus
						// a victim endpoint marker; everything else is a point.
						if (hasKillLine(event)) {
							const start = toLatLng(event.attackerX as number, event.attackerY as number);
							const end = toLatLng(event.eventX as number, event.eventY as number);
							return (
								<Fragment key={event.matchEventId}>
									<Polyline positions={[start, end]} pathOptions={{ color, weight: 2, opacity: 0.9 }}>
										<Tooltip>{event.summary}</Tooltip>
										{popup}
									</Polyline>
									<CircleMarker
										center={end}
										radius={3}
										pathOptions={{ color, fillColor: color, fillOpacity: 1, weight: 1 }}
										interactive={false}
									/>
								</Fragment>
							);
						}
						return (
							<CircleMarker
								key={event.matchEventId}
								center={toLatLng(event.eventX as number, event.eventY as number)}
								radius={event.eventType.startsWith("building_") ? 5 : 3.5}
								pathOptions={{ color, fillColor: color, fillOpacity: 0.9, weight: 1.5 }}
							>
								<Tooltip>{event.summary}</Tooltip>
								{popup}
							</CircleMarker>
						);
					})}
				</MapContainer>
			</Box>
			<Box sx={{ padding: 1.5 }}>
				<Typography variant="caption" color="textSecondary" sx={{ textTransform: "uppercase" }}>
					Timeline scrub
				</Typography>
				<Stack direction="row" spacing={2} sx={{ alignItems: "center", marginBottom: 1 }}>
					<Box sx={{ flexGrow: 1 }}>
						<Slider
							min={minTick}
							max={maxTick}
							step={1}
							value={scrubTick}
							onChange={(_e, value) => setTickFilter(value as number)}
							valueLabelDisplay="auto"
							valueLabelFormat={(value) => `Tick ${value}`}
							aria-label="Scrub events by demo tick"
						/>
					</Box>
					<Typography variant="body2" sx={{ minWidth: 220 }}>
						{filteredEvents.length > 0
							? `Tick ${scrubTick} · ${formatMatchClock(filteredEvents[filteredEvents.length - 1].ticksSinceStart)} · ${filteredEvents.length} event${filteredEvents.length === 1 ? "" : "s"}`
							: `Tick ${scrubTick} · no events yet`}
					</Typography>
				</Stack>
				<Typography variant="caption" color="textSecondary" sx={{ textTransform: "uppercase" }}>
					Layers — showing {visibleEvents.length} of {filteredEvents.length} positioned events
				</Typography>
				<Stack direction="row" spacing={2} useFlexGap sx={{ flexWrap: "wrap", marginBottom: 1.5 }}>
					{layerKeys.map((layerKey) => (
						<FormControlLabel
							key={layerKey}
							control={
								<Switch
									checked={eventVisibility[layerKey] !== false}
									onChange={(e) =>
										setEventVisibility((prev) => ({ ...prev, [layerKey]: e.target.checked }))
									}
								/>
							}
							label={
								<Stack direction="row" spacing={0.75} sx={{ alignItems: "center" }}>
									<Box
										sx={{
											width: 12,
											height: 12,
											borderRadius: "50%",
											backgroundColor: eventTypeColor(layerKey),
										}}
									/>
									<Typography variant="body2">
										{eventMapLayerLabel(layerKey)} ({countsByLayer.get(layerKey) ?? 0})
									</Typography>
								</Stack>
							}
						/>
					))}
					{referenceMarkers.length > 0 && (
						<FormControlLabel
							control={
								<Switch
									checked={showReferencePoints}
									onChange={(event) => setShowReferencePoints(event.target.checked)}
								/>
							}
							label={`Reference points (${referenceMarkers.length})`}
						/>
					)}
				</Stack>
			</Box>
		</ContainerWithHeaderAndButtons>
	);
};

/** Deprecated alias kept while callers migrate to the event map. */
export const MatchKillMap = MatchEventMap;
