/** biome-ignore-all lint/correctness/noChildrenProp: ts-form made me do it! */
import { CloudDownload } from "@mui/icons-material";
import FlagIcon from "@mui/icons-material/Flag";
import { IconButton, Link } from "@mui/material";
import Grid from "@mui/material/Grid";
import Tooltip from "@mui/material/Tooltip";
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
import { apiGetDemos, apiGetServers } from "../api";
import { IconButtonLink } from "../component/IconButtonLink.tsx";
import { RowActionContainer } from "../component/RowActionContainer.tsx";
import { TextLink } from "../component/TextLink.tsx";
import {
	createDefaultTableOptions,
	makeRowActionsDefOptions,
	makeSchemaState,
	type OnChangeFn,
	setColumnFilter,
} from "../component/table/options.ts";
import { SortableTable } from "../component/table/SortableTable.tsx";
import { TableCellRelativeDateField } from "../component/table/TableCellRelativeDateField.tsx";
import { useAuth } from "../hooks/useAuth.ts";
import type { DemoFile } from "../schema/demo.ts";
import { stringToColour } from "../util/colours.ts";
import { ensureFeatureEnabled } from "../util/features.ts";
import { humanFileSize } from "../util/text.tsx";

const columnHelper = createMRTColumnHelper<DemoFile>();
const defaultOptions = createDefaultTableOptions<DemoFile>();
const validateSearch = makeSchemaState("created_on");

export const Route = createFileRoute("/_guest/stv")({
	component: STV,
	validateSearch,
	beforeLoad: ({ context }) => {
		ensureFeatureEnabled(context.appInfo.demos_enabled);
	},
	loader: async ({ context }) => {
		const unsorted = await context.queryClient.ensureQueryData({
			queryKey: ["serversSimple"],
			queryFn: async ({ signal }) => {
				return await apiGetServers(signal);
			},
		});

		return {
			servers: unsorted.sort((a, b) =>
				a.server_name > b.server_name ? 1 : a.server_name < b.server_name ? -1 : 0,
			),
		};
	},
	head: ({ match }) => ({
		meta: [
			{
				name: "description",
				content: "Search and download SourceTV recordings",
			},
			match.context.title("SourceTV"),
		],
	}),
});

function STV() {
	const { isAuthenticated } = useAuth();
	const { servers } = Route.useLoaderData();
	const navigate = useNavigate();
	const search = Route.useSearch();

	const { data, isLoading, isError } = useQuery({
		queryKey: ["demos"],
		queryFn: async ({ signal }) => {
			return await apiGetDemos(signal);
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
	const columns = useMemo(() => {
		return [
			columnHelper.accessor("demo_id", {
				header: "ID",
				grow: false,
				Cell: ({ cell }) => <Typography>#{cell.getValue()}</Typography>,
			}),
			columnHelper.accessor("server_id", {
				filterFn: (row, _, filterValue) => {
					return filterValue.length === 0 || filterValue.includes(row.original.server_id);
				},
				filterVariant: "multi-select",
				filterSelectOptions: servers.map((server) => ({
					label: server.server_name,
					value: server.server_id,
				})),
				grow: false,
				enableSorting: true,
				enableColumnFilter: true,
				header: "Server",
				Cell: ({ row, cell }) => (
					<Tooltip title={row.original.server_name_long}>
						<TextLink
							to={"/stv"}
							search={setColumnFilter(search, "server_id", [cell.getValue()])}
							sx={{ color: stringToColour(row.original.server_name_short) }}
						>
							{row.original.server_name_short}
						</TextLink>
					</Tooltip>
				),
			}),
			columnHelper.accessor("created_on", {
				header: "Created",
				enableColumnFilter: false,
				enableSorting: true,
				filterVariant: "date",
				grow: false,
				Cell: ({ cell }) => <TableCellRelativeDateField date={cell.getValue()} suffix />,
			}),
			columnHelper.accessor("map_name", {
				enableColumnFilter: true,
				header: "Map Name",
				grow: true,
				Cell: ({ row, cell }) => (
					<TextLink
						to={"/stv"}
						search={setColumnFilter(search, "map_name", cell.getValue())}
						sx={{ color: stringToColour(row.original.map_name) }}
					>
						{row.original.map_name}
					</TextLink>
				),
			}),
			columnHelper.accessor("size", {
				header: "Size",
				enableColumnFilter: false,
				enableSorting: false,
				grow: false,
				Cell: ({ cell }) => <Typography>{humanFileSize(cell.getValue() as number)}</Typography>,
			}),
			columnHelper.accessor("stats", {
				header: "SteamID",
				grow: false,
				enableSorting: false,
				enableColumnFilter: true,
				filterFn: (row, _, filterValue) => {
					return filterValue === "" || Object.keys(row.original.stats).includes(filterValue);
				},
				Cell: ({ cell }) => <Typography>{Object.keys(Object(cell.getValue())).length} Players</Typography>,
			}),
		];
	}, [servers, search]);

	const table = useMaterialReactTable({
		...defaultOptions,
		columns,
		data: data ?? [],
		enableFilters: true,
		enableHiding: true,
		enableFacetedValues: true,
		onColumnFiltersChange: setColumnFilters,
		onPaginationChange: setPagination,
		onSortingChange: setSorting,
		displayColumnDefOptions: makeRowActionsDefOptions(2),
		state: {
			isLoading,
			showAlertBanner: isError,
			columnFilters: search.columnFilters,
			pagination: search.pagination,
			sorting: search.sorting,
		},
		initialState: {
			...defaultOptions.initialState,
			columnVisibility: {
				demo_id: false,
				server_id: true,
				created_on: true,
			},
		},
		enableRowActions: true,
		renderRowActions: ({ row }) => (
			<RowActionContainer>
				<IconButtonLink
					key={"report"}
					disabled={!isAuthenticated()}
					color={"error"}
					to={"/report"}
					search={{ demo_id: row.original.demo_id }}
				>
					<FlagIcon />
				</IconButtonLink>
				<IconButton component={Link} key={"dl-link"} color={"success"} href={`/asset/${row.original.asset_id}`}>
					<CloudDownload />
				</IconButton>
			</RowActionContainer>
		),
	});
	return (
		<Grid container>
			<Grid size={{ xs: 12 }}>
				<SortableTable table={table} title={"SourceTV Recordings"} />
			</Grid>
		</Grid>
	);
}
