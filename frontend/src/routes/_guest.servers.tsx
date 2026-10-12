import { useQuery } from "@connectrpc/connect-query";
import ChevronRightIcon from "@mui/icons-material/ChevronRight";
import ContentCopyIcon from "@mui/icons-material/ContentCopy";
import FilterListIcon from "@mui/icons-material/FilterList";
import Button from "@mui/material/Button";
import FormControl from "@mui/material/FormControl";
import FormControlLabel from "@mui/material/FormControlLabel";
import Grid from "@mui/material/Grid";
import IconButton from "@mui/material/IconButton";
import InputLabel from "@mui/material/InputLabel";
import Link from "@mui/material/Link";
import MenuItem from "@mui/material/MenuItem";
import Paper from "@mui/material/Paper";
import Select, { type SelectChangeEvent } from "@mui/material/Select";
import Stack from "@mui/material/Stack";
import Switch from "@mui/material/Switch";
import Tooltip from "@mui/material/Tooltip";
import Typography from "@mui/material/Typography";
import { createFileRoute, stripSearchParams, useNavigate } from "@tanstack/react-router";
import L, { type LatLngLiteral } from "leaflet";
import * as markerIcon from "leaflet/dist/images/marker-icon.png";
import * as markerIcon2x from "leaflet/dist/images/marker-icon-2x.png";
import * as markerShadow from "leaflet/dist/images/marker-shadow.png";
import { createMRTColumnHelper, type MRT_ColumnDef, useMaterialReactTable } from "material-react-table";
import { type ChangeEvent, useCallback, useEffect, useMemo, useState } from "react";
import { MapContainer, TileLayer, useMap } from "react-leaflet";
import z from "zod/v4";
import { ContainerWithHeader } from "../component/ContainerWithHeader.tsx";
import { Flag } from "../component/Flag.tsx";
import { RangeSlider } from "../component/RangeSlider.tsx";
import { ServerMarkers } from "../component/ServerMarkers.tsx";
import { createDefaultTableOptions } from "../component/table/options.ts";
import { SortableTable } from "../component/table/SortableTable.tsx";
import { UserPingRadius } from "../component/UserPingRadius.tsx";
import { UserPositionMarker } from "../component/UserPositionMarker.tsx";
import { MapStateCtx } from "../contexts/MapStateCtx.tsx";
import { useMapStateCtx } from "../hooks/useMapStateCtx.ts";
import { useUserFlashCtx } from "../hooks/useUserFlashCtx.ts";
import type { SafeServer } from "../rpc/servers/v1/servers_pb.ts";
import { state } from "../rpc/servers/v1/servers-ServersService_connectquery.ts";
import { tf2Fonts } from "../theme.ts";
import { logErr } from "../util/errors.ts";
import { ensureFeatureEnabled } from "../util/features.ts";
import { uniqCI } from "../util/lists.ts";
import { cleanMapName } from "../util/strings.ts";

// Workaround for leaflet not loading icons properly in react
// eslint-disable-next-line @typescript-eslint/ban-ts-comment
// @ts-expect-error
delete L.Icon.Default.prototype._getIconUrl;
L.Icon.Default.mergeOptions({
	iconRetinaUrl: markerIcon2x.default,
	iconUrl: markerIcon.default,
	shadowUrl: markerShadow.default,
});

export const Route = createFileRoute("/_guest/servers")({
	component: Servers,
	validateSearch: z.object({
		selectedRegion: z.string().optional().default("any"),
		filterByRegion: z.boolean().optional().default(false),
		showOpenOnly: z.boolean().optional().default(false),
		customRange: z.number().optional().default(1500),
	}),
	search: {
		middlewares: [
			stripSearchParams({
				selectedRegion: "any",
				filterByRegion: false,
				showOpenOnly: false,
				customRange: 1500,
			}),
		],
	},
	beforeLoad: ({ context }) => {
		ensureFeatureEnabled(context.appInfo.serversEnabled);
	},
	head: ({ match }) => ({
		meta: [{ name: "description", content: "Server Browser" }, match.context.title("Servers")],
	}),
});

const columnHelper = createMRTColumnHelper<SafeServer>();
const defaultOptions = createDefaultTableOptions<SafeServer>();

// MapContainer only honours `center` on mount, so recenter explicitly when
// the geoip position arrives.
function RecenterMap() {
	const map = useMap();
	const { pos } = useMapStateCtx();

	useEffect(() => {
		if (pos.lat !== 0 || pos.lng !== 0) {
			map.setView(pos, map.getZoom());
		}
	}, [map, pos]);

	return null;
}

function Servers() {
	const search = Route.useSearch();
	const navigate = useNavigate();
	const [pos, setPos] = useState<LatLngLiteral>({
		lat: 0.0,
		lng: 0.0,
	});
	const [selectedServers, setSelectedServers] = useState<SafeServer[]>([]);
	const { customRange, filterByRegion, showOpenOnly, selectedRegion } = search;
	const { sendFlash } = useUserFlashCtx();
	const { data, isLoading } = useQuery(state, {}, { refetchInterval: 5000 });
	const regions = uniqCI(["any", ...(data?.servers || []).map((value) => value.region)]);

	// Filter state lives in the URL so filtered views are shareable. Replace
	// (rather than push) history entries so slider drags don't spam history.
	const setFilterSearch = useCallback(
		(patch: Partial<typeof search>) => {
			navigate({
				to: Route.fullPath,
				search: (prev) => ({ ...prev, ...patch }),
				replace: true,
			});
		},
		[navigate],
	);

	const setCustomRange = useCallback(
		(value: number) => {
			setFilterSearch({ customRange: value });
		},
		[setFilterSearch],
	);

	const setFilterByRegion = useCallback(
		(value: boolean) => {
			setFilterSearch({ filterByRegion: value });
		},
		[setFilterSearch],
	);

	const setShowOpenOnly = useCallback(
		(value: boolean) => {
			setFilterSearch({ showOpenOnly: value });
		},
		[setFilterSearch],
	);

	const setSelectedRegion = useCallback(
		(value: string) => {
			setFilterSearch({ selectedRegion: value });
		},
		[setFilterSearch],
	);

	// Center the map (and the "you" marker) on the geoip position reported
	// by the backend once it arrives.
	useEffect(() => {
		const lat = data?.latLong?.latitude ?? 0;
		const lng = data?.latLong?.longitude ?? 0;
		if ((lat !== 0 || lng !== 0) && (lat !== pos.lat || lng !== pos.lng)) {
			setPos({ lat, lng });
		}
	}, [data, pos]);

	const onRegionsChange = (event: SelectChangeEvent) => {
		setSelectedRegion(event.target.value);
	};

	const onShowOpenOnlyChanged = (_: ChangeEvent<HTMLInputElement>, checked: boolean) => {
		setShowOpenOnly(checked);
	};

	const onRegionsToggleEnabledChanged = (_: ChangeEvent<HTMLInputElement>, checked: boolean) => {
		setFilterByRegion(checked);
	};

	useEffect(() => {
		let s = (data?.servers ?? []).sort((a, b) => {
			// Sort by position if we have a non-default position.
			// otherwise, sort by server name
			if (pos.lat !== 0) {
				return a.distance > b.distance ? 1 : a.distance < b.distance ? -1 : 0;
			}
			return `${a.nameShort}`.localeCompare(b.nameShort);
		});
		if (!filterByRegion && !selectedRegion.includes("any")) {
			s = s.filter((srv) => selectedRegion.includes(srv.region));
		}
		if (showOpenOnly) {
			s = s.filter((srv) => (srv?.players || 0) < (srv?.maxPlayers || 32));
		}
		if (filterByRegion && customRange && customRange > 0) {
			s = s.filter((srv) => srv.distance < customRange);
		}
		setSelectedServers(s);
	}, [selectedRegion, showOpenOnly, filterByRegion, customRange, pos, data?.servers]);

	const marks = [
		{
			value: 500,
			label: "500 km",
		},
		{
			value: 1500,
			label: "1500 km",
		},
		{
			value: 3000,
			label: "3000 km",
		},
		{
			value: 5000,
			label: "5000 km",
		},
	];

	const metaServers = useMemo(() => {
		return selectedServers.map((s) => ({ ...s, copy: "", connect: "" }));
	}, [selectedServers]);

	const columns = useMemo(
		() =>
			[
				columnHelper.accessor("cc", {
					header: "CC",
					size: 40,
					Cell: ({ cell }) => <Flag countryCode={cell.getValue()} />,
				}),
				columnHelper.accessor("name", {
					header: "Server",
					size: 450,
					Cell: ({ cell }) => (
						<Typography variant={"button"} sx={{ fontFamily: tf2Fonts }}>
							{cell.getValue()}
						</Typography>
					),
				}),
				columnHelper.accessor("map", {
					header: "Map",
					size: 150,
					Cell: ({ cell }) => <Typography variant={"body2"}>{cleanMapName(cell.getValue())}</Typography>,
				}),
				columnHelper.accessor("players", {
					header: "Players",
					size: 50,
					Cell: ({ row }) => (
						<Typography
							variant={"body2"}
						>{`${row.original.players}/${Number(row.original.maxPlayers) > 0 ? row.original.maxPlayers : row.original.maxPlayers}`}</Typography>
					),
				}),
				columnHelper.accessor("distance", {
					header: "Dist",

					size: 60,
					meta: {
						tooltip: "Approximate distance from you",
					},
					Cell: ({ cell }) => (
						<Tooltip title={`Distance in hammer units: ${Math.round((cell.getValue() ?? 1) * 52.49)} khu`}>
							<Typography variant={"caption"}>{`${cell.getValue().toFixed(0)}km`}</Typography>
						</Tooltip>
					),
				}),
				columnHelper.display({
					header: "Cp",
					size: 30,
					meta: {
						tooltip: "Copy to clipboard",
					},
					Cell: ({ row }) => (
						<IconButton
							color={"primary"}
							aria-label={"Copy connect string to clipboard"}
							onClick={() => {
								navigator.clipboard
									.writeText(`connect ${row.original.ip}:${row.original.port}`)
									.then(() => {
										sendFlash("success", "Copied address to clipboard");
									})
									.catch((e) => {
										sendFlash("error", "Failed to copy address");
										logErr(e);
									});
							}}
						>
							<ContentCopyIcon />
						</IconButton>
					),
				}),
				columnHelper.display({
					header: "Connect",
					size: 125,
					Cell: ({ row }) => (
						<Button
							fullWidth
							endIcon={<ChevronRightIcon />}
							component={Link}
							href={`steam://run/440//+connect ${row.original.ip}:${row.original.port}`}
							variant={"contained"}
							sx={{ minWidth: 100 }}
						>
							Join
						</Button>
					),
				}),
			].filter((f) => f) as Array<MRT_ColumnDef<SafeServer>>,
		[sendFlash],
	);
	const table = useMaterialReactTable({
		...defaultOptions,
		columns: columns,
		data: metaServers ?? [],
		enableFilters: false,
		enableColumnFilters: false,
		enableSorting: false,
		enableRowActions: false,
		enableColumnActions: false,
		enablePagination: false,
		initialState: {
			...defaultOptions.initialState,
			pagination: {
				pageSize: 100,
				pageIndex: 0,
			},
			sorting: [{ id: "distance", desc: true }],
			columnVisibility: {
				name: true,
			},
		},
	});
	return (
		<MapStateCtx.Provider
			value={{
				servers: isLoading ? [] : (data?.servers ?? []),
				customRange,
				setCustomRange,
				pos,
				setPos,
				selectedServers,
				setSelectedServers,
				filterByRegion,
				setFilterByRegion,
				showOpenOnly,
				setShowOpenOnly,
				selectedRegion,
				setSelectedRegion,
			}}
		>
			<Stack spacing={3}>
				<Paper elevation={3}>
					<MapContainer
						center={pos}
						zoom={3}
						scrollWheelZoom={true}
						id={"map"}
						style={{ height: "500px", width: "100%" }}
						attributionControl={true}
						minZoom={3}
						worldCopyJump={true}
					>
						<TileLayer
							url="https://{s}.tile.openstreetmap.org/{z}/{x}/{y}.png"
							attribution={"© OpenStreetMap contributors "}
						/>

						<UserPingRadius />
						<ServerMarkers servers={selectedServers} />
						<UserPositionMarker />
						<RecenterMap />
					</MapContainer>
				</Paper>
				<ContainerWithHeader title={"Filters"} iconLeft={<FilterListIcon />}>
					<Grid
						container
						spacing={2}
						style={{
							width: "100%",
							flexWrap: "nowrap",
							alignItems: "center",
							padding: 10,
							// justifyContent: 'center'
						}}
					>
						<Grid size={{ xs: 2 }}>
							<FormControlLabel
								control={
									<Switch checked={showOpenOnly} onChange={onShowOpenOnlyChanged} name="checkedA" />
								}
								label="Open Slots"
							/>
						</Grid>
						<Grid size={{ xs: 2 }}>
							<FormControl>
								<InputLabel id="region-selector-label">Region</InputLabel>
								<Select<string>
									disabled={filterByRegion}
									labelId="region-selector-label"
									id="region-selector"
									value={selectedRegion}
									onChange={onRegionsChange}
								>
									{regions.map((r) => {
										return (
											<MenuItem key={`region-${r}`} value={r}>
												{r}
											</MenuItem>
										);
									})}
								</Select>
							</FormControl>
						</Grid>
						<Grid size={{ xs: 2 }}>
							<FormControlLabel
								control={
									<Switch
										checked={filterByRegion}
										onChange={onRegionsToggleEnabledChanged}
										name="regionsEnabled"
									/>
								}
								label="By Range"
							/>
						</Grid>
						<Grid size={{ xs: 6 }} style={{ paddingRight: "2rem" }}>
							<RangeSlider
								style={{
									zIndex: 1000,
								}}
								disabled={!filterByRegion}
								defaultValue={1000}
								aria-labelledby="custom-range"
								step={100}
								max={5000}
								valueLabelDisplay="off"
								value={customRange}
								marks={marks}
								onChange={(_: Event, value: number | number[]) => {
									setCustomRange(value as number);
								}}
							/>
						</Grid>
					</Grid>
				</ContainerWithHeader>
				<SortableTable table={table} hideToolbarButtons />
			</Stack>
		</MapStateCtx.Provider>
	);
}
