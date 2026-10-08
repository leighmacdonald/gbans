import "leaflet/dist/leaflet.css";
import MapIcon from "@mui/icons-material/Map";
import Alert from "@mui/material/Alert";
import Box from "@mui/material/Box";
import FormControlLabel from "@mui/material/FormControlLabel";
import MenuItem from "@mui/material/MenuItem";
import Paper from "@mui/material/Paper";
import Select from "@mui/material/Select";
import Stack from "@mui/material/Stack";
import Switch from "@mui/material/Switch";
import Typography from "@mui/material/Typography";
import { CRS } from "leaflet";
import { useEffect, useMemo, useState } from "react";
import { CircleMarker, ImageOverlay, MapContainer, Polyline, Popup, Tooltip } from "react-leaflet";
import { useMap } from "react-leaflet/hooks";
import { Team } from "../../rpc/stats/v1/stats_pb.ts";
import { formatMatchClock } from "../../util/time.ts";
import { ContainerWithHeaderAndButtons } from "../ContainerWithHeaderAndButtons.tsx";
import {
	DIMMED_OPACITY,
	fetchReferenceFile,
	fetchWorldFile,
	HIGHLIGHT_DEATH,
	HIGHLIGHT_KILL,
	overviewPixelBounds,
	overviewUrls,
	paddedOverviewPixelBounds,
	pixelBoundsToLatLng,
	type ReferencePoint,
	referencePointLabel,
	referencePointStyle,
	referenceUrls,
	teamColorOf,
	WORLD_COLOR,
	type WorldFile,
	worldToPixel,
} from "./killMap.ts";
import type { KillFeedEntry, MatchView } from "./match";

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

const isWorldKill = (kill: KillFeedEntry): boolean => kill.killerSteamId === "" || kill.killerSteamId === "0";

export const MatchKillMap = ({ summary, mapName }: { summary: MatchView; mapName: string }) => {
	const [worldFile, setWorldFile] = useState<WorldFile | null>(null);
	const [imgSize, setImgSize] = useState<{ width: number; height: number } | null>(null);
	const [referencePoints, setReferencePoints] = useState<ReferencePoint[]>([]);
	const [loadError, setLoadError] = useState<string | null>(null);
	const [selectedSteamId, setSelectedSteamId] = useState<string | null>(null);
	const [roundFilter, setRoundFilter] = useState<number>(0);
	const [showKillLines, setShowKillLines] = useState(true);
	const [showKillMarkers, setShowKillMarkers] = useState(true);
	const [showReferencePoints, setShowReferencePoints] = useState(true);

	useEffect(() => {
		let cancelled = false;
		setWorldFile(null);
		setImgSize(null);
		setReferencePoints([]);
		setLoadError(null);
		if (summary.kills.length === 0) {
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
	}, [mapName, summary.kills.length]);

	const teams = useMemo(() => {
		const out: Record<string, Team> = {};
		for (const s of summary.summaries) {
			out[s.player.steamId] = s.team;
		}
		return out;
	}, [summary.summaries]);

	const nameOf = (steamId: string): string => {
		if (steamId === "" || steamId === "0") {
			return "World";
		}
		return summary.players[steamId]?.name ?? steamId;
	};

	const rounds = useMemo(() => {
		const seen = new Set<number>();
		for (const k of summary.kills) {
			if (k.round > 0) {
				seen.add(k.round);
			}
		}
		return [...seen].toSorted((a, b) => a - b);
	}, [summary.kills]);

	const roundLabel = (round: number): string => {
		const r = summary.rounds.find((x) => x.round === round);
		return r ? `Round ${round} (${r.scoreBlu} – ${r.scoreRed})` : `Round ${round}`;
	};

	const kills = useMemo(
		() => (roundFilter === 0 ? summary.kills : summary.kills.filter((k) => k.round === roundFilter)),
		[summary.kills, roundFilter],
	);

	const legend = useMemo(() => {
		const counts = new Map<string, number>();
		for (const k of kills) {
			if (!isWorldKill(k)) {
				counts.set(k.killerSteamId, (counts.get(k.killerSteamId) ?? 0) + 1);
			}
		}
		return [...counts.entries()].toSorted((a, b) => b[1] - a[1]);
	}, [kills]);

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

	if (summary.kills.length === 0) {
		return (
			<Paper sx={{ padding: 2 }}>
				<Alert severity="info">No kill events were recorded for this match.</Alert>
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

	const baseColor = (kill: KillFeedEntry): string => {
		if (isWorldKill(kill)) {
			return WORLD_COLOR;
		}
		return teamColorOf(teams[kill.killerSteamId] ?? Team.UNASSIGNED_UNSPECIFIED);
	};

	const toggleSelect = (steamId: string) => {
		if (steamId === "" || steamId === "0") {
			return;
		}
		setSelectedSteamId((prev) => (prev === steamId ? null : steamId));
	};

	const roundFilterControl = (
		<Stack key="kill-map-round-filter" direction="row" spacing={1} sx={{ alignItems: "center" }}>
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
		<ContainerWithHeaderAndButtons title="Kill Map" iconLeft={<MapIcon />} buttons={[roundFilterControl]} padding={0} spacing={0}>
			<Box sx={{ paddingX: 1.5, paddingTop: 1.5 }}>
				<MapContainer
					crs={CRS.Simple}
					center={[imgSize.height / 2, imgSize.width / 2]}
					zoom={0}
					minZoom={-5}
					maxZoom={3}
					zoomSnap={0.25}
					zoomDelta={0.25}
					maxBounds={viewLatLngBounds}
					maxBoundsViscosity={1}
					scrollWheelZoom={false}
					style={{ height: MAP_HEIGHT, width: "100%" }}
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
					{showKillLines &&
						kills.map((kill) => {
							const start = toLatLng(kill.killerX, kill.killerY);
							const end = toLatLng(kill.victimX, kill.victimY);
							const isKiller = selectedSteamId !== null && kill.killerSteamId === selectedSteamId;
							const isVictim = selectedSteamId !== null && kill.victimSteamId === selectedSteamId;
							const color = isKiller ? HIGHLIGHT_KILL : isVictim ? HIGHLIGHT_DEATH : baseColor(kill);
							const dimmed = selectedSteamId !== null && !isKiller && !isVictim;
							return (
								<Polyline
									key={kill.matchKillId}
									positions={[start, end]}
									pathOptions={{
										color,
										weight: isKiller || isVictim ? 4 : 2,
										opacity: dimmed ? DIMMED_OPACITY : 0.9,
									}}
									eventHandlers={{ click: () => toggleSelect(kill.killerSteamId) }}
								>
									<Popup>
										<Stack spacing={0.5}>
											<Typography variant="body2" sx={{ fontWeight: 700 }}>
												{nameOf(kill.killerSteamId)} → {nameOf(kill.victimSteamId)}
											</Typography>
											<Typography variant="caption">
												{kill.weapon} · Round {kill.round > 0 ? kill.round : "—"} ·{" "}
												{formatMatchClock(kill.ticksSinceStart)}
											</Typography>
										</Stack>
									</Popup>
								</Polyline>
							);
						})}
					{showKillMarkers &&
						kills.map((kill) => {
							const color =
								selectedSteamId !== null && kill.killerSteamId === selectedSteamId
									? HIGHLIGHT_KILL
									: selectedSteamId !== null && kill.victimSteamId === selectedSteamId
										? HIGHLIGHT_DEATH
										: baseColor(kill);
							return (
								<CircleMarker
									key={`m-${kill.matchKillId}`}
									center={toLatLng(kill.killerX, kill.killerY)}
									radius={3}
									pathOptions={{ color, fillColor: color, fillOpacity: 1, weight: 1 }}
									interactive={false}
								/>
							);
						})}
				</MapContainer>
			</Box>
			<Box sx={{ padding: 1.5 }}>
				<Typography variant="caption" color="textSecondary" sx={{ textTransform: "uppercase" }}>
					Layers
				</Typography>
				<Stack direction="row" spacing={2} useFlexGap sx={{ flexWrap: "wrap", marginBottom: 1.5 }}>
					<FormControlLabel
						control={
							<Switch
								checked={showKillLines}
								onChange={(event) => setShowKillLines(event.target.checked)}
							/>
						}
						label={`Kill lines (${kills.length})`}
					/>
					<FormControlLabel
						control={
							<Switch
								checked={showKillMarkers}
								onChange={(event) => setShowKillMarkers(event.target.checked)}
							/>
						}
						label={`Killer markers (${kills.length})`}
					/>
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
				<Typography variant="caption" color="textSecondary" sx={{ textTransform: "uppercase" }}>
					Killers — click a name or a line to highlight kills and deaths
				</Typography>
				<Stack direction="row" spacing={1} useFlexGap sx={{ flexWrap: "wrap", marginTop: 0.5 }}>
					{legend.map(([steamId, count]) => {
						const selected = selectedSteamId === steamId;
						return (
							<Box
								key={steamId}
								onClick={() => toggleSelect(steamId)}
								title={`${nameOf(steamId)}: ${count} kills`}
								sx={{
									display: "flex",
									alignItems: "center",
									gap: 0.75,
									paddingX: 1,
									paddingY: 0.5,
									borderRadius: 1,
									cursor: "pointer",
									border: "1px solid",
									borderColor: selected ? HIGHLIGHT_KILL : "divider",
									backgroundColor: selected ? "action.selected" : "transparent",
								}}
							>
								<Box
									sx={{
										width: 12,
										height: 12,
										borderRadius: "50%",
										backgroundColor: teamColorOf(teams[steamId] ?? Team.UNASSIGNED_UNSPECIFIED),
									}}
								/>
								<Typography variant="body2">
									{nameOf(steamId)} ({count})
								</Typography>
							</Box>
						);
					})}
				</Stack>
			</Box>
		</ContainerWithHeaderAndButtons>
	);
};
