import Grid from "@mui/material/Grid";
import Typography from "@mui/material/Typography";
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
import { z } from "zod/v4";
import { apiGetConnections, apiGetServers } from "../api";
import { TextLink } from "../component/TextLink.tsx";
import { createDefaultTableOptions, makeSchemaState, type OnChangeFn } from "../component/table/options.ts";
import { SortableTable } from "../component/table/SortableTable.tsx";
import type { PersonConnection } from "../schema/people.ts";
import { renderDateTime } from "../util/time.ts";
import "leaflet/dist/leaflet.css";
import { useTheme } from "@mui/material";
import Paper from "@mui/material/Paper";
import Tooltip from "@mui/material/Tooltip";
import L from "leaflet";
import * as markerIcon from "leaflet/dist/images/marker-icon.png";
import * as markerIcon2x from "leaflet/dist/images/marker-icon-2x.png";
import * as markerShadow from "leaflet/dist/images/marker-shadow.png";
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

const validateSearch = z
	.object({
		cidr: z.cidrv4().optional(),
		source_id: z.string().optional(),
		as_num: z.number().optional(),
		as_name: z.string().optional(),
		country_code: z.string().optional(),
		country_name: z.string().optional(),
	})
	.extend(makeSchemaState({ defaultSortColumn: "person_connection_id" }));

export const Route = createFileRoute("/_mod/admin/network/playersbyip")({
	component: AdminNetworkPlayersByCIDR,
	validateSearch,
	loader: async ({ context }) => {
		const unsorted = await context.queryClient.ensureQueryData({
			queryKey: ["serversSimple"],
			queryFn: apiGetServers,
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
	const theme = useTheme();

	const { data, isLoading, isError, isRefetching } = useQuery({
		queryKey: ["playersByIP", { search }],
		queryFn: async () => {
			const server_id = search.columnFilters?.find((filter) => filter.id === "server_id")?.value;
			const cidr = String(search.columnFilters?.find((filter) => filter.id === "ip_addr")?.value ?? "");
			const as_name = String(search.columnFilters?.find((filter) => filter.id === "as_name")?.value ?? "");
			const as_num = Number(search.columnFilters?.find((filter) => filter.id === "as_num")?.value ?? 0);
			const city_name = String(search.columnFilters?.find((filter) => filter.id === "city_name")?.value ?? "");
			const country_code = String(
				search.columnFilters?.find((filter) => filter.id === "country_code")?.value ?? "",
			);
			const country_name = String(
				search.columnFilters?.find((filter) => filter.id === "country_name")?.value ?? "",
			);
			const source_id = String(search.columnFilters?.find((filter) => filter.id === "steam_id")?.value ?? "");
			const sort = search.sorting?.find((sort) => sort);

			return await apiGetConnections({
				desc: sort ? sort.desc : true,
				limit: search.pagination?.pageSize,
				offset: search.pagination ? search.pagination.pageIndex * search.pagination?.pageSize : 0,
				order_by: sort ? sort.id : "person_connection_id",
				source_id,
				server_id: Number(server_id) > 0 ? [Number(server_id)] : [],
				cidr,
				as_name,
				as_num,
				city_name,
				country_code,
				country_name,
			});
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
				Cell: ({ row }) => (
					<Tooltip title={row.original.server_name}>
						<Typography sx={{ color: stringToColour(row.original.server_name ?? "", theme.palette.mode) }}>
							{row.original.server_name_short}
						</Typography>
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
					<TextLink to={"/profile/$steamId"} params={{ steamId: cell.getValue() }}>
						{cell.getValue()}
					</TextLink>
				),
			}),
			columnHelper.accessor("ip_addr", {
				grow: true,
				header: "IP",
			}),
			columnHelper.accessor("as_num", {
				id: "as_num",
				header: "AS Num",
				grow: false,
				enableSorting: false,
			}),
			columnHelper.accessor("as_name", {
				header: "AS Name",
				grow: true,
				enableSorting: false,
			}),
			columnHelper.accessor("country_code", {
				header: "Country",
				grow: true,
				enableSorting: false,
			}),
			columnHelper.accessor("city_name", {
				header: "City Name",
				grow: true,
				enableSorting: false,
			}),
			columnHelper.accessor("lat_long", {
				header: "Lat Long",
				grow: true,
				enableSorting: false,
				Cell: ({ cell }) => `${cell.getValue().latitude} / ${cell.getValue().longitude}`,
			}),
		],
		[servers, theme.palette.mode],
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
		manualFiltering: true,
		manualPagination: true,
		manualSorting: true,
		onColumnFiltersChange: setColumnFilters,
		onPaginationChange: setPagination,
		onSortingChange: setSorting,
		enableRowActions: true,
		enableCellActions: true,
		// muiTableBodyCellProps: () => {
		// 	return {
		// 		onClick: async (event) => {
		// 			console.log(event);
		// 			await navigate({
		// 				to: "/admin/network/playersbyip",
		// 				search: s,
		// 			});
		// 		},
		// 		sx: {
		// 			cursor: "pointer", //you might want to change the cursor too when adding an onClick
		// 		},
		// 	};
		// },
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
				<SortableTable
					table={table}
					title={"Network Connections (Brought to you with our partners at Palantir)"}
				/>
			</Grid>
		</Grid>
	);
}
