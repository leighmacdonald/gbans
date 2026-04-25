/** biome-ignore-all lint/correctness/noChildrenProp: ts-form made me do it! */

import { type Timestamp, timestampDate } from "@bufbuild/protobuf/wkt";
import { useQuery } from "@connectrpc/connect-query";
import { CloudDownload } from "@mui/icons-material";
import FlagIcon from "@mui/icons-material/Flag";
import { IconButton, Link } from "@mui/material";
import Grid from "@mui/material/Grid";
import Tooltip from "@mui/material/Tooltip";
import Typography from "@mui/material/Typography";
import { createFileRoute, stripSearchParams, useNavigate } from "@tanstack/react-router";
import {
	createMRTColumnHelper,
	type MRT_ColumnFiltersState,
	type MRT_PaginationState,
	type MRT_SortingState,
	useMaterialReactTable,
} from "material-react-table";
import { useCallback, useMemo } from "react";
import { IconButtonLink } from "../component/IconButtonLink.tsx";
import { RowActionContainer } from "../component/RowActionContainer.tsx";
import { TextLink } from "../component/TextLink.tsx";
import {
	createDefaultTableOptions,
	makeRowActionsDefOptions,
	makeSchemaDefaults,
	makeSchemaState,
	type OnChangeFn,
	setColumnFilter,
} from "../component/table/options.ts";
import { SortableTable } from "../component/table/SortableTable.tsx";
import { TableCellRelativeDateField } from "../component/table/TableCellRelativeDateField.tsx";
import { useAuth } from "../hooks/useAuth.ts";
import type { Demo } from "../rpc/demo/v1/demo_pb.ts";
import { getDemos } from "../rpc/demo/v1/demo-DemoService_connectquery.ts";
import { servers } from "../rpc/servers/v1/servers-ServersService_connectquery.ts";
import { stringToColour } from "../util/colours.ts";
import { ensureFeatureEnabled } from "../util/features.ts";
import { humanFileSize } from "../util/text.tsx";

const columnHelper = createMRTColumnHelper<Demo>();
const defaultOptions = createDefaultTableOptions<Demo>();
const defaultValues = makeSchemaDefaults({ defaultColumn: "created_on" });
const validateSearch = makeSchemaState("created_on");

export const Route = createFileRoute("/_guest/stv")({
	component: STV,
	validateSearch,
	search: {
		middlewares: [stripSearchParams(defaultValues)],
	},
	beforeLoad: ({ context }) => {
		ensureFeatureEnabled(context.appInfo.demosEnabled);
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
	const navigate = useNavigate();
	const search = Route.useSearch();

	const { data: serversList, isLoading: isLoadingServers } = useQuery(servers);

	const { data, isLoading, isError } = useQuery(getDemos);

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
			columnHelper.accessor("demoId", {
				header: "ID",
				grow: false,
				Cell: ({ cell }) => <Typography>#{cell.getValue()}</Typography>,
			}),
			columnHelper.accessor("serverId", {
				filterFn: (row, _, filterValue) => {
					return filterValue.length === 0 || filterValue.includes(row.original.serverId);
				},
				filterVariant: "multi-select",
				filterSelectOptions: serversList?.servers.map((server) => ({
					label: server.serverName,
					value: server.serverId,
				})),
				grow: false,
				enableSorting: true,
				enableColumnFilter: true,
				header: "Server",
				Cell: ({ row, cell }) => (
					<Tooltip title={row.original.serverNameLong}>
						<TextLink
							to={"/stv"}
							search={setColumnFilter(search, "server_id", [cell.getValue()])}
							sx={{ color: stringToColour(row.original.serverNameShort) }}
						>
							{row.original.serverNameShort}
						</TextLink>
					</Tooltip>
				),
			}),
			columnHelper.accessor("createdOn", {
				header: "Created",
				enableColumnFilter: false,
				enableSorting: true,
				filterVariant: "date",
				grow: false,
				Cell: ({ cell }) => (
					<TableCellRelativeDateField date={timestampDate(cell.getValue() as Timestamp)} suffix />
				),
			}),
			columnHelper.accessor("mapName", {
				enableColumnFilter: true,
				header: "Map Name",
				grow: true,
				Cell: ({ row, cell }) => (
					<TextLink
						to={"/stv"}
						search={setColumnFilter(search, "map_name", cell.getValue())}
						sx={{ color: stringToColour(row.original.mapName) }}
					>
						{row.original.mapName}
					</TextLink>
				),
			}),
			columnHelper.accessor("size", {
				header: "Size",
				enableColumnFilter: false,
				enableSorting: false,
				grow: false,
				Cell: ({ cell }) => <Typography>{humanFileSize(Number(cell.getValue()))}</Typography>,
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
	}, [serversList?.servers, search]);

	const table = useMaterialReactTable({
		...defaultOptions,
		columns,
		data: data?.demos ?? [],
		enableFilters: true,
		enableHiding: true,
		enableFacetedValues: true,
		onColumnFiltersChange: setColumnFilters,
		onPaginationChange: setPagination,
		onSortingChange: setSorting,
		displayColumnDefOptions: makeRowActionsDefOptions(2),
		state: {
			isLoading: isLoading || isLoadingServers,
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
					search={{ demo_id: Number(row.original.demoId) }}
				>
					<FlagIcon />
				</IconButtonLink>
				<IconButton component={Link} key={"dl-link"} color={"success"} href={`/asset/${row.original.assetId}`}>
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
