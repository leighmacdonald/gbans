import DnsIcon from "@mui/icons-material/Dns";
import Grid from "@mui/material/Grid";
import { createFileRoute, stripSearchParams, useNavigate } from "@tanstack/react-router";
import {
	createMRTColumnHelper,
	type MRT_ColumnFiltersState,
	type MRT_PaginationState,
	type MRT_SortingState,
	useMaterialReactTable,
} from "material-react-table";
import { useCallback, useMemo } from "react";
import { CircleMarker, MapContainer, Marker, Popup, TileLayer } from "react-leaflet";
import { TextLink } from "../component/TextLink.tsx";
import {
	createDefaultTableOptions,
	makeSchemaDefaults,
	makeSchemaState,
	type OnChangeFn,
	setColumnFilter,
} from "../component/table/options.ts";
import { SortableTable } from "../component/table/SortableTable.tsx";
import { renderTimestamp } from "../util/time.ts";
import "leaflet/dist/leaflet.css";
import { useQuery } from "@connectrpc/connect-query";
import { useTheme } from "@mui/material";
import Paper from "@mui/material/Paper";
import Tooltip from "@mui/material/Tooltip";
import L from "leaflet";
import * as markerIcon from "leaflet/dist/images/marker-icon.png";
import * as markerIcon2x from "leaflet/dist/images/marker-icon-2x.png";
import * as markerShadow from "leaflet/dist/images/marker-shadow.png";
import { IconButtonLink } from "../component/IconButtonLink.tsx";
import { RowActionContainer } from "../component/RowActionContainer.tsx";
import type { PersonConnection } from "../rpc/network/v1/network_pb.ts";
import { queryConnections } from "../rpc/network/v1/network-NetworkService_connectquery.ts";
import { servers } from "../rpc/servers/v1/servers-ServersService_connectquery.ts";
import { stringToColour } from "../util/colours.ts";

// Workaround for leaflet not loading icons properly in react
// eslint-disable-next-line @typescript-eslint/ban-ts-comment
// @ts-expect-error
delete L.Icon.Default.prototype._getIconUrl;
L.Icon.Default.mergeOptions({
	iconRetinaUrl: markerIcon2x.default,
	iconUrl: markerIcon.default,
	shadowUrl: markerShadow.default,
});

const columnHelper = createMRTColumnHelper<PersonConnection>();
const defaultOptions = createDefaultTableOptions<PersonConnection>();
const defaultValues = makeSchemaDefaults({ defaultColumn: "person_connection_id" });
const validateSearch = makeSchemaState("person_connection_id");

export const Route = createFileRoute("/_mod/admin/network/playersbyip")({
	component: AdminNetworkPlayersByCIDR,
	validateSearch,
	search: {
		middlewares: [stripSearchParams(defaultValues)],
	},
	head: ({ match }) => ({
		meta: [{ name: "description", content: "Find players by IP address" }, match.context.title("Players By IP")],
	}),
});

function AdminNetworkPlayersByCIDR() {
	const { data: serversList } = useQuery(servers);
	const search = Route.useSearch();
	const navigate = useNavigate();
	const theme = useTheme();

	const { data, isLoading, isError, isRefetching } = useQuery(queryConnections, {});
	// {
	// 	queryKey: ["playersByIP", { search }],
	// 	queryFn: async ({ signal }) => {
	// 		const server_id = filterValue<PersonConnection>("server_id", search.columnFilters);
	// 		const sort = search.sorting ? sortValueDefault(search.sorting, "person_connection_id") : undefined;
	//
	// 		return await apiGetConnections(
	// 			{
	// 				desc: sort ? sort.desc : true,
	// 				limit: search.pagination?.pageSize,
	// 				offset: search.pagination ? search.pagination.pageIndex * search.pagination?.pageSize : 0,
	// 				order_by: sort ? sort.id : "person_connection_id",
	// 				source_id: filterValue<PersonConnection>("steam_id", search.columnFilters),
	// 				server_id: Number(server_id) > 0 ? [Number(server_id)] : [],
	// 				cidr: filterValue<PersonConnection>("ip_addr", search.columnFilters),
	// 				as_name: filterValue<PersonConnection>("as_name", search.columnFilters),
	// 				as_num: filterValueNumber<PersonConnection>("as_num", search.columnFilters),
	// 				city_name: filterValue("city_name", search.columnFilters),
	// 				country_code: filterValue("country_code", search.columnFilters),
	// 				country_name: filterValue("country_name", search.columnFilters),
	// 			},
	// 			signal,
	// 		);
	// 	},
	// });

	const setSorting: OnChangeFn<MRT_SortingState> = useCallback(
		(updater) => {
			navigate({
				to: Route.fullPath,
				search: {
					...search,
					sorting: typeof updater === "function" ? updater(search.sorting ?? []) : updater,
				},
			});
		},
		[search, navigate],
	);

	const setColumnFilters: OnChangeFn<MRT_ColumnFiltersState> = useCallback(
		(updater) => {
			navigate({
				to: Route.fullPath,
				search: {
					...search,
					columnFilters: typeof updater === "function" ? updater(search.columnFilters ?? []) : updater,
				},
			});
		},
		[search, navigate],
	);

	const setPagination: OnChangeFn<MRT_PaginationState> = useCallback(
		(updater) => {
			navigate({
				to: Route.fullPath,
				search: {
					...search,
					pagination:
						typeof updater === "function"
							? updater(search.pagination ?? { pageIndex: 0, pageSize: 50 })
							: updater,
				},
			});
		},
		[search, navigate],
	);

	const columns = useMemo(
		() => [
			columnHelper.accessor("serverId", {
				header: "Server",
				grow: false,
				enableSorting: false,
				filterVariant: "multi-select",
				filterSelectOptions: serversList?.servers.map((server) => ({
					label: server.serverName,
					value: server.serverId,
				})),
				filterFn: (row, _, filterValue) => {
					return filterValue.length === 0 || filterValue.includes(row.original.serverId);
				},
				Cell: ({ row, cell }) => (
					<Tooltip title={row.original.serverId}>
						<TextLink
							to={"/admin/network/playersbyip"}
							search={setColumnFilter(search, "server_id", [cell.getValue()])}
							sx={{ color: stringToColour(row.original.serverNameShort ?? "") }}
						>
							{row.original.serverNameShort}
						</TextLink>
					</Tooltip>
				),
			}),
			columnHelper.accessor("createdOn", {
				grow: false,
				enableSorting: true,
				enableColumnFilter: false,
				filterVariant: "date-range",
				header: "Created",
				Cell: ({ cell }) => renderTimestamp(cell.getValue()),
			}),
			columnHelper.accessor("personaName", {
				header: "Name",
				grow: false,
				enableSorting: false,
			}),
			columnHelper.accessor("steamId", {
				grow: false,
				header: "Steam ID",
				enableSorting: false,
				Cell: ({ cell }) => (
					<TextLink
						color={theme.palette.mode === "dark" ? theme.palette.primary.light : theme.palette.primary.dark}
						to={"/admin/network/playersbyip"}
						search={setColumnFilter(search, "steam_id", cell.getValue())}
					>
						{cell.getValue()}
					</TextLink>
				),
			}),
			columnHelper.accessor("ipAddr", {
				grow: true,
				header: "IP",
				Cell: ({ cell }) => (
					<TextLink
						to={"/admin/network/playersbyip"}
						search={setColumnFilter(search, "ip_addr", [cell.getValue()])}
					>
						{cell.getValue()}
					</TextLink>
				),
			}),
			columnHelper.accessor("asNum", {
				id: "as_num",
				header: "AS Num",
				grow: false,
				enableSorting: false,
				Cell: ({ cell }) => (
					<TextLink to={Route.fullPath} search={setColumnFilter(search, "as_num", cell.getValue())}>
						{cell.getValue()}
					</TextLink>
				),
			}),
			columnHelper.accessor("asName", {
				header: "AS Name",
				grow: true,
				enableSorting: false,
				Cell: ({ cell }) => (
					<TextLink to={Route.fullPath} search={setColumnFilter(search, "as_name", cell.getValue())}>
						{cell.getValue()}
					</TextLink>
				),
			}),
			columnHelper.accessor("countryCode", {
				header: "Country",
				grow: true,
				enableSorting: false,
				Cell: ({ cell }) => (
					<TextLink to={Route.fullPath} search={setColumnFilter(search, "country_code", cell.getValue())}>
						{cell.getValue()}
					</TextLink>
				),
			}),
			columnHelper.accessor("cityName", {
				header: "City Name",
				grow: true,
				enableSorting: false,
				Cell: ({ cell }) => (
					<TextLink to={Route.fullPath} search={setColumnFilter(search, "city_name", cell.getValue())}>
						{cell.getValue()}
					</TextLink>
				),
			}),
			columnHelper.accessor("location", {
				header: "Lat Long",
				grow: true,
				enableSorting: false,
				Cell: ({ row }) => `${row.original.location?.latitude} / ${row.original.location?.longitude}`,
			}),
		],
		[search, theme, serversList?.servers.map],
	);

	const table = useMaterialReactTable({
		...defaultOptions,
		paginationDisplayMode: "pages",
		columns,
		data: data?.connection ?? [],
		//pageCount: -1,
		//rowCount: data?.count ?? undefined,
		enableFilters: true,
		enableHiding: true,
		enableFacetedValues: true,
		enableRowVirtualization: true,
		paginateExpandedRows: true,
		state: {
			isLoading: isLoading || isRefetching,
			showAlertBanner: isError,
			showProgressBars: isRefetching,
			pagination: search.pagination,
			sorting: search.sorting,
			columnFilters: search.columnFilters,
		},
		initialState: {
			...defaultOptions.initialState,
			columnVisibility: {
				person_connection_id: false,
				server_id: true,
				address: true,
				created_on: true,
				updated_on: false,
				lat_long: false,
			},
		},
		onColumnFiltersChange: setColumnFilters,
		onPaginationChange: setPagination,
		onSortingChange: setSorting,
		enableRowActions: true,
		enableCellActions: true,
		renderRowActions: ({ row }) => (
			<RowActionContainer>
				<Tooltip title={"IP Information"} key={1}>
					<IconButtonLink
						color={"info"}
						to={"/admin/network/ipInfo"}
						search={{
							ip: row.original.ipAddr,
						}}
					>
						<DnsIcon />
					</IconButtonLink>
				</Tooltip>
			</RowActionContainer>
		),
	});

	return (
		<Grid container spacing={2}>
			<Grid size={{ xs: 12 }}>
				<Paper>
					<MapContainer
						center={{ lat: 51.505, lng: -0.09 }}
						zoom={3}
						scrollWheelZoom={true}
						id={"ip-map"}
						style={{ height: "500px", width: "100%" }}
						attributionControl={true}
						minZoom={3}
						worldCopyJump={true}
					>
						<TileLayer
							url="https://{s}.tile.openstreetmap.org/{z}/{x}/{y}.png"
							attribution={"© OpenStreetMap contributors "}
							detectRetina={true}
						/>
						{table.getRowModel().rows.map((row) =>
							row.getIsSelected() ? (
								<CircleMarker
									key={row.id}
									color="red"
									center={
										row.original.location
											? {
													lat: row.original.location.latitude,
													lng: row.original.location.longitude,
												}
											: { lat: 42.4338, lng: 83.9845 }
									}
								/>
							) : (
								row.original.location && (
									<Marker
										key={row.id}
										position={{
											lat: row.original.location.latitude,
											lng: row.original.location.longitude,
										}}
									>
										<Popup>{JSON.stringify(row.original)}</Popup>
									</Marker>
								)
							),
						)}
					</MapContainer>
				</Paper>
			</Grid>
			<Grid size={{ xs: 12 }}>
				<SortableTable table={table} title={"Player Connection History"} />
			</Grid>
		</Grid>
	);
}
