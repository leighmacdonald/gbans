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
import { createFileRoute } from "@tanstack/react-router";
import L, { type LatLngLiteral } from "leaflet";
import * as markerIcon from "leaflet/dist/images/marker-icon.png";
import * as markerIcon2x from "leaflet/dist/images/marker-icon-2x.png";
import * as markerShadow from "leaflet/dist/images/marker-shadow.png";
import { createMRTColumnHelper, type MRT_ColumnDef, useMaterialReactTable } from "material-react-table";
import { type ChangeEvent, useCallback, useEffect, useMemo, useState } from "react";
import { MapContainer, TileLayer } from "react-leaflet";
import { ContainerWithHeader } from "../component/ContainerWithHeader.tsx";
import { Flag } from "../component/Flag.tsx";
import { RangeSlider } from "../component/RangeSlider.tsx";
import { ServerMarkers } from "../component/ServerMarkers.tsx";
import { createDefaultTableOptions } from "../component/table/options.ts";
import { SortableTable } from "../component/table/SortableTable.tsx";
import { UserPingRadius } from "../component/UserPingRadius.tsx";
import { UserPositionMarker } from "../component/UserPositionMarker.tsx";
import { MapStateCtx } from "../contexts/MapStateCtx.tsx";
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
	beforeLoad: ({ context }) => {
		ensureFeatureEnabled(context.appInfo.serversEnabled);
	},
	head: ({ match }) => ({
		meta: [{ name: "description", content: "Server Browser" }, match.context.title("Servers")],
	}),
});

const columnHelper = createMRTColumnHelper<SafeServer>();
const defaultOptions = createDefaultTableOptions<SafeServer>();

function Servers() {
	const [pos, setPos] = useState<LatLngLiteral>({
		lat: 0.0,
		lng: 0.0,
	});
	const [customRange, setCustomRange] = useState<number>(500);
	const [selectedServers, setSelectedServers] = useState<SafeServer[]>([]);
	const [filterByRegion, setFilterByRegion] = useState<boolean>(false);
	const [showOpenOnly, setShowOpenOnly] = useState<boolean>(false);
	const [selectedRegion, setSelectedRegion] = useState<string>("any");
	const { sendFlash } = useUserFlashCtx();
	const { data, isLoading } = useQuery(state, {}, { refetchInterval: 5000 });
	const regions = uniqCI(["any", ...(data?.servers || []).map((value) => value.region)]);

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
		const defaultState = {
			showOpenOnly: false,
			selectedRegion: "any",
			filterByRegion: false,
			customRange: 1500,
		};
		let state = defaultState;
		try {
			const val = localStorage.getItem("filters");
			if (val) {
				state = JSON.parse(val);
			}
		} catch (e) {
			logErr(e);
			return;
		}
		setShowOpenOnly(state?.showOpenOnly || defaultState.showOpenOnly);
		setSelectedRegion(state?.selectedRegion !== "" ? state.selectedRegion : defaultState.selectedRegion);
		setFilterByRegion(state?.filterByRegion || defaultState.filterByRegion);
		setCustomRange(state?.customRange || defaultState.customRange);
	}, []);

	const saveFilterState = useCallback(() => {
		localStorage.setItem(
			"filters",
			JSON.stringify({
				showOpenOnly: showOpenOnly,
				selectedRegion: selectedRegion,
				filterByRegion: filterByRegion,
				customRange: customRange,
			}),
		);
	}, [customRange, filterByRegion, selectedRegion, showOpenOnly]);

	useEffect(() => {
		let s = (data?.servers ?? []).sort((a, b) => {
			// Sort by position if we have a non-default position.
			// otherwise, sort by server name
			if (pos.lat !== 0) {
				if (a.distance > b.distance) {
					return 1;
				}
				if (a.distance < b.distance) {
					return -1;
				}
				return 0;
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
		saveFilterState();
	}, [selectedRegion, showOpenOnly, filterByRegion, customRange, saveFilterState, pos, data?.servers]);

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

	// const { restart } = useTimer({
	// 	autoStart: true,
	// 	expiryTimestamp: new Date(),
	// 	onExpire: () => {
	// 		// TODO replace this with tan query
	// 		const ac = new AbortController();
	// 		apiGetServerStates(ac.signal)
	// 			.then((response) => {
	// 				if (!response) {
	// 					restart(nextExpiry());
	// 					return;
	// 				}
	// 				setServers(response.servers || []);
	// 				if (pos.lat === 0) {
	// 					setPos({
	// 						lat: response.lat_long.latitude,
	// 						lng: response.lat_long.longitude,
	// 					});
	// 				}
	//
	// 				restart(nextExpiry());
	// 			})
	// 			.catch(() => {
	// 				restart(nextExpiry());
	// 			});
	// 	},
	// });
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
						<Typography variant={"button"} fontFamily={tf2Fonts}>
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
				<SortableTable table={table} title={"Servers"} hideToolbarButtons />
			</Stack>
		</MapStateCtx.Provider>
	);
}
