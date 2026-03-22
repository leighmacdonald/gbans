import DnsIcon from "@mui/icons-material/Dns";
import Grid from "@mui/material/Grid";
import { useQuery } from "@tanstack/react-query";
import { createFileRoute, useNavigate } from "@tanstack/react-router";
import {
	createMRTColumnHelper,
	type MRT_ColumnFiltersState,
	type MRT_PaginationState,
	type MRT_SortingState,
	useMaterialReactTable,
} from "material-react-table";
import { useCallback, useMemo } from "react";
import { CircleMarker, MapContainer, Marker, Popup, TileLayer } from "react-leaflet";
import { apiGetConnections, apiGetServers } from "../api";
import { TextLink } from "../component/TextLink.tsx";
import {
	createDefaultTableOptions,
	filterValue,
	filterValueNumber,
	makeSchemaState,
	type OnChangeFn,
	setColumnFilter,
	sortValueDefault,
} from "../component/table/options.ts";
import { SortableTable } from "../component/table/SortableTable.tsx";
import type { PersonConnection } from "../schema/people.ts";
import { renderDateTime } from "../util/time.ts";
import "leaflet/dist/leaflet.css";
import Paper from "@mui/material/Paper";
import Tooltip from "@mui/material/Tooltip";
import L from "leaflet";
import * as markerIcon from "leaflet/dist/images/marker-icon.png";
import * as markerIcon2x from "leaflet/dist/images/marker-icon-2x.png";
import * as markerShadow from "leaflet/dist/images/marker-shadow.png";
import { IconButtonLink } from "../component/IconButtonLink.tsx";
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

const validateSearch = makeSchemaState("person_connection_id");

export const Route = createFileRoute("/_mod/admin/network/playersbyip")({
	component: AdminNetworkPlayersByCIDR,
	validateSearch,
	loader: async ({ context }) => {
		const unsorted = await context.queryClient.ensureQueryData({
			queryKey: ["serversSimple"],
			queryFn: async ({ signal }) => {
				return await apiGetServers(signal);
			},
		});
		return unsorted.sort((a, b) => {
			return a.server_name > b.server_name ? 1 : a.server_name < b.server_name ? -1 : 0;
		});
	},
	head: ({ match }) => ({
		meta: [{ name: "description", content: "Find players by IP address" }, match.context.title("Players By IP")],
	}),
});

function AdminNetworkPlayersByCIDR() {
	const servers = Route.useLoaderData();
	const search = Route.useSearch();
	const navigate = useNavigate();

	const { data, isLoading, isError, isRefetching } = useQuery({
		queryKey: ["playersByIP", { search }],
		queryFn: async ({ signal }) => {
			const server_id = filterValue<PersonConnection>("server_id", search.columnFilters);
			const sort = search.sorting ? sortValueDefault(search.sorting, "person_connection_id") : undefined;

			return await apiGetConnections(
				{
					desc: sort ? sort.desc : true,
					limit: search.pagination?.pageSize,
					offset: search.pagination ? search.pagination.pageIndex * search.pagination?.pageSize : 0,
					order_by: sort ? sort.id : "person_connection_id",
					source_id: filterValue<PersonConnection>("steam_id", search.columnFilters),
					server_id: Number(server_id) > 0 ? [Number(server_id)] : [],
					cidr: filterValue<PersonConnection>("ip_addr", search.columnFilters),
					as_name: filterValue<PersonConnection>("as_name", search.columnFilters),
					as_num: filterValueNumber<PersonConnection>("as_num", search.columnFilters),
					city_name: filterValue("city_name", search.columnFilters),
					country_code: filterValue("country_code", search.columnFilters),
					country_name: filterValue("country_name", search.columnFilters),
				},
				signal,
			);
		},
	});

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
			columnHelper.accessor("server_id", {
				header: "Server",
				grow: false,
				enableSorting: false,
				filterVariant: "multi-select",
				filterSelectOptions: servers.map((server) => ({
					label: server.server_name,
					value: server.server_id,
				})),
				filterFn: (row, _, filterValue) => {
					return filterValue.length === 0 || filterValue.includes(row.original.server_id);
				},
				Cell: ({ row, cell }) => (
					<Tooltip title={row.original.server_name}>
						<TextLink
							to={"/admin/network/playersbyip"}
							search={setColumnFilter(search, "server_id", [cell.getValue()])}
							sx={{ color: stringToColour(row.original.server_name ?? "") }}
						>
							{row.original.server_name_short}
						</TextLink>
					</Tooltip>
				),
			}),
			columnHelper.accessor("created_on", {
				grow: false,
				enableSorting: true,
				enableColumnFilter: false,
				filterVariant: "date-range",
				header: "Created",
				Cell: ({ cell }) => renderDateTime(cell.getValue()),
			}),
			columnHelper.accessor("persona_name", {
				header: "Name",
				grow: false,
				enableSorting: false,
			}),
			columnHelper.accessor("steam_id", {
				grow: false,
				header: "Steam ID",
				enableSorting: false,
				Cell: ({ cell }) => (
					<TextLink
						to={"/admin/network/playersbyip"}
						search={setColumnFilter(search, "steam_id", cell.getValue())}
					>
						{cell.getValue()}
					</TextLink>
				),
			}),
			columnHelper.accessor("ip_addr", {
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
			columnHelper.accessor("as_num", {
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
			columnHelper.accessor("as_name", {
				header: "AS Name",
				grow: true,
				enableSorting: false,
				Cell: ({ cell }) => (
					<TextLink to={Route.fullPath} search={setColumnFilter(search, "as_name", cell.getValue())}>
						{cell.getValue()}
					</TextLink>
				),
			}),
			columnHelper.accessor("country_code", {
				header: "Country",
				grow: true,
				enableSorting: false,
				Cell: ({ cell }) => (
					<TextLink to={Route.fullPath} search={setColumnFilter(search, "country_code", cell.getValue())}>
						{cell.getValue()}
					</TextLink>
				),
			}),
			columnHelper.accessor("city_name", {
				header: "City Name",
				grow: true,
				enableSorting: false,
				Cell: ({ cell }) => (
					<TextLink to={Route.fullPath} search={setColumnFilter(search, "city_name", cell.getValue())}>
						{cell.getValue()}
					</TextLink>
				),
			}),
			columnHelper.accessor("lat_long", {
				header: "Lat Long",
				grow: true,
				enableSorting: false,
				Cell: ({ cell }) => `${cell.getValue().latitude} / ${cell.getValue().longitude}`,
			}),
		],
		[servers, search],
	);

	const table = useMaterialReactTable({
		...defaultOptions,
		paginationDisplayMode: "pages",
		columns,
		data: data?.data ?? [],
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
		renderRowActionMenuItems: ({ row }) => [
			<Tooltip title={"IP Information"} key={1}>
				<IconButtonLink
					color={"info"}
					to={"/admin/network/ipInfo"}
					search={{
						ip: row.original.ip_addr,
					}}
				>
					<DnsIcon />
				</IconButtonLink>
			</Tooltip>,
		],
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
										row.original.lat_long
											? {
													lat: row.original.lat_long.latitude,
													lng: row.original.lat_long.longitude,
												}
											: { lat: 42.4338, lng: 83.9845 }
									}
								/>
							) : (
								row.original.lat_long && (
									<Marker
										key={row.id}
										position={{
											lat: row.original.lat_long.latitude,
											lng: row.original.lat_long.longitude,
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
