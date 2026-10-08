import "leaflet/dist/leaflet.css";
import Alert from "@mui/material/Alert";
import Box from "@mui/material/Box";
import MenuItem from "@mui/material/MenuItem";
import Paper from "@mui/material/Paper";
import Select from "@mui/material/Select";
import Stack from "@mui/material/Stack";
import Typography from "@mui/material/Typography";
import { CRS } from "leaflet";
import { useEffect, useMemo, useState } from "react";
import { CircleMarker, ImageOverlay, MapContainer, Polyline, Popup } from "react-leaflet";
import { useMap } from "react-leaflet/hooks";
import { Team } from "../../rpc/stats/v1/stats_pb.ts";
import { formatMatchClock } from "../../util/time.ts";
import {
	DIMMED_OPACITY,
	fetchWorldFile,
	HIGHLIGHT_DEATH,
	HIGHLIGHT_KILL,
	overviewUrls,
	teamColorOf,
	WORLD_COLOR,
	type WorldFile,
	worldToPixel,
} from "./killMap.ts";
import type { KillFeedEntry, MatchView } from "./match";

const MAP_HEIGHT = 480;

const clamp = (v: number, lo: number, hi: number): number => Math.min(hi, Math.max(lo, v));

const FitOverview = ({ width, height }: { width: number; height: number }) => {
	const map = useMap();
	useEffect(() => {
		map.fitBounds([
			[0, 0],
			[height, width],
		]);
	}, [map, width, height]);
	return null;
};

const isWorldKill = (kill: KillFeedEntry): boolean => kill.killerSteamId === "" || kill.killerSteamId === "0";

export const MatchKillMap = ({ summary, mapName }: { summary: MatchView; mapName: string }) => {
	const [worldFile, setWorldFile] = useState<WorldFile | null>(null);
	const [imgSize, setImgSize] = useState<{ width: number; height: number } | null>(null);
	const [loadError, setLoadError] = useState<string | null>(null);
	const [selectedSteamId, setSelectedSteamId] = useState<string | null>(null);
	const [roundFilter, setRoundFilter] = useState<number>(0);

	useEffect(() => {
		let cancelled = false;
		setWorldFile(null);
		setImgSize(null);
		setLoadError(null);
		if (summary.kills.length === 0) {
			return;
		}
		const urls = overviewUrls(mapName);
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

	if (!worldFile || !imgSize) {
		return (
			<Paper sx={{ padding: 2 }}>
				<Typography variant="body2" color="textSecondary">
					Loading overview…
				</Typography>
			</Paper>
		);
	}

	const toLatLng = (x: number, y: number): [number, number] => {
		const { px, py } = worldToPixel(worldFile, x, y);
		return [imgSize.height - clamp(py, 0, imgSize.height), clamp(px, 0, imgSize.width)];
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

	return (
		<Paper>
			<Stack
				direction={{ xs: "column", sm: "row" }}
				spacing={1}
				sx={{ padding: 1.5, alignItems: { sm: "center" }, justifyContent: "space-between" }}
			>
				<Typography variant="h6" sx={{ fontWeight: 900 }}>
					Kill Map
				</Typography>
				<Stack direction="row" spacing={1} sx={{ alignItems: "center" }}>
					<Typography variant="body2" color="textSecondary">
						Round
					</Typography>
					<Select
						size="small"
						value={roundFilter}
						onChange={(e) => setRoundFilter(Number(e.target.value))}
						sx={{ minWidth: 160 }}
					>
						<MenuItem value={0}>All rounds</MenuItem>
						{rounds.map((r) => (
							<MenuItem key={r} value={r}>
								{roundLabel(r)}
							</MenuItem>
						))}
					</Select>
				</Stack>
			</Stack>
			<Box sx={{ paddingX: 1.5 }}>
				<MapContainer
					crs={CRS.Simple}
					center={[imgSize.height / 2, imgSize.width / 2]}
					zoom={0}
					scrollWheelZoom={false}
					style={{ height: MAP_HEIGHT, width: "100%" }}
				>
					<FitOverview width={imgSize.width} height={imgSize.height} />
					<ImageOverlay
						url={overviewUrls(mapName).image}
						bounds={[
							[0, 0],
							[imgSize.height, imgSize.width],
						]}
					/>
					{kills.map((kill) => {
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
					{kills.map((kill) => {
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
		</Paper>
	);
};
