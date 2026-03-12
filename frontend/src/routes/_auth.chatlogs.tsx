import RefreshIcon from "@mui/icons-material/Refresh";
import ReportIcon from "@mui/icons-material/Report";
import { IconButton, TableCell, Typography, useTheme } from "@mui/material";
import Grid from "@mui/material/Grid";
import Tooltip from "@mui/material/Tooltip";
import { keepPreviousData, useQuery } from "@tanstack/react-query";
import { createFileRoute } from "@tanstack/react-router";
import {
	createMRTColumnHelper,
	type MRT_ColumnFiltersState,
	type MRT_PaginationState,
	type MRT_SortingState,
	useMaterialReactTable,
} from "material-react-table";
import { useMemo, useState } from "react";
import { apiGetMessages, apiGetServers } from "../api";
import { IconButtonLink } from "../component/IconButtonLink.tsx";
import { PersonCell } from "../component/PersonCell.tsx";
import { createDefaultTableOptions } from "../component/table/options.ts";
import { SortableTable } from "../component/table/SortableTable.tsx";
import { TableCellRelativeDateField } from "../component/table/TableCellRelativeDateField.tsx";
import type { PersonMessage } from "../schema/people.ts";
import { stringToColour } from "../util/colours.ts";
import { ensureFeatureEnabled } from "../util/features.ts";

export const Route = createFileRoute("/_auth/chatlogs")({
	component: ChatLogs,
	beforeLoad: ({ context }) => {
		ensureFeatureEnabled(context.appInfo.chatlogs_enabled);
	},
	loader: async ({ context }) => {
		const unsorted = await context.queryClient.ensureQueryData({
			queryKey: ["serversSimple"],
			queryFn: apiGetServers,
		});
		return {
			servers: unsorted.sort((a, b) => {
				if (a.server_name > b.server_name) {
					return 1;
				}
				if (a.server_name < b.server_name) {
					return -1;
				}
				return 0;
			}),
		};
	},
	head: ({ match }) => ({
		meta: [{ name: "description", content: "Browse in-game chat logs" }, match.context.title("Chat Logs")],
	}),
});

const columnHelper = createMRTColumnHelper<PersonMessage>();
const defaultOptions = createDefaultTableOptions<PersonMessage>();

function ChatLogs() {
	const [columnFilters, setColumnFilters] = useState<MRT_ColumnFiltersState>([]);
	const [globalFilter, setGlobalFilter] = useState("");
	const [sorting, setSorting] = useState<MRT_SortingState>([]);
	const [pagination, setPagination] = useState<MRT_PaginationState>({
		pageIndex: 0,
		pageSize: 50,
	});
	const theme = useTheme();
	// const { hasPermission } = useAuth();
	const { servers } = Route.useLoaderData();
	console.log(servers);
	// const navigate = useNavigate({ from: Route.fullPath });
	const columns = useMemo(() => {
		return [
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
					console.log(filterValue);
					return filterValue.length === 0 || filterValue.includes(row.original.server_id);
				},
				size: 125,
				Cell: ({ row }) => (
					<TableCell
						sx={{
							color: stringToColour(row.original.server_name, theme.palette.mode),
						}}
					>
						{row.original.server_name}
					</TableCell>
				),
			}),

			columnHelper.accessor("created_on", {
				header: "Created",
				enableColumnFilter: false,
				grow: false,
				size: 150,
				Cell: (info) => <TableCellRelativeDateField date={info.row.original.created_on} />,
			}),

			columnHelper.accessor("steam_id", {
				header: "SteamID",
				grow: true,
				enableSorting: false,
				enableColumnFilter: true,
				filterFn: (row, _, filterValue) => {
					const query = filterValue.toLowerCase();
					if (query === "") {
						return true;
					}
					const value = row.original.steam_id.toLowerCase();
					if (value.includes(query)) {
						return true;
					}
					if (row.original.steam_id.includes(query) || row.original.steam_id === query) {
						return true;
					}

					return false;
				},
				Cell: ({ row }) => (
					<PersonCell
						showCopy={true}
						steam_id={row.original.steam_id}
						avatar_hash={row.original.avatar_hash}
						personaname={row.original.persona_name}
					/>
				),
			}),

			columnHelper.accessor("body", {
				header: "Message",
				grow: true,
				enableSorting: false,
				Cell: ({ cell }) => (
					<Typography padding={0} variant={"body1"}>
						{cell.getValue()}
					</Typography>
				),
			}),
			// columnHelper.accessor("auto_filter_flagged", {
			// 	header: "Flag",
			// 	grow: false,
			// 	size: 30,
			// 	Cell: ({ cell }) =>
			// 		cell.getValue() > 0 ? (
			// 			<Tooltip title={"Message already flagged"}>
			// 				<FlagIcon color={"error"} />
			// 			</Tooltip>
			// 		) : null,
			// }),
			columnHelper.display({
				header: "Flag",
				grow: false,
				size: 60,
				enableSorting: false,
				Cell: ({ row }) => {
					return (
						<Tooltip title={"Create Report"}>
							<IconButtonLink
								color={"error"}
								disabled={row.original.auto_filter_flagged > 0}
								to={"/report"}
								search={{
									person_message_id: row.original.person_message_id,
									steam_id: row.original.steam_id,
								}}
							>
								<ReportIcon />
							</IconButtonLink>
						</Tooltip>
					);
				},
			}),
		];
	}, [theme.palette.mode, servers]);

	const { data, isLoading, isError, isRefetching, refetch } = useQuery({
		queryKey: ["chatlogs", { columnFilters, globalFilter, pagination, sorting }],
		queryFn: async () => {
			const server_id = columnFilters.find((filter) => filter.id === "server_id")?.value;
			const steam_id = columnFilters.find((filter) => filter.id === "steam_id")?.value;
			const body = columnFilters.find((filter) => filter.id === "body")?.value;
			const sort = sorting.find((sort) => sort);

			return await apiGetMessages({
				server_id: server_id ? Number(server_id) : 0,
				personaname: "",
				query: body ? String(body) : "",
				source_id: steam_id ? String(steam_id) : "",
				limit: pagination.pageSize,
				offset: pagination.pageIndex * pagination.pageSize,
				order_by: sort ? sort.id : "person_message_id",
				desc: sort ? sort.desc : false,
				flagged_only: false,
			});
		},
		//refetchInterval: search.auto_refresh,
		placeholderData: keepPreviousData,
	});

	const table = useMaterialReactTable({
		...defaultOptions,
		columns,
		data: data ? data.data : [],
		rowCount: data ? data.count : 0,
		enableFilters: true,
		state: {
			columnFilters,
			globalFilter,
			isLoading,
			pagination,
			showAlertBanner: isError,
			showProgressBars: isRefetching,
			sorting,
		},
		initialState: {
			...defaultOptions.initialState,
			sorting: [{ id: "updated_on", desc: true }],
			columnVisibility: {
				server_id: true,
				source_id: true,
				body: true,
				created_on: true,
			},
		},
		manualFiltering: true,
		manualPagination: true,
		manualSorting: true,
		muiToolbarAlertBannerProps: isError
			? {
					color: "error",
					children: "Error loading data",
				}
			: undefined,
		onColumnFiltersChange: setColumnFilters,
		onGlobalFilterChange: setGlobalFilter,
		onPaginationChange: setPagination,
		onSortingChange: setSorting,
		renderTopToolbarCustomActions: () => (
			<Tooltip arrow title="Refresh Data">
				<IconButton onClick={() => refetch()}>
					<RefreshIcon />
				</IconButton>
			</Tooltip>
		),
	});

	return (
		<Grid container spacing={2}>
			<Grid size={{ xs: 12 }}>
				<SortableTable table={table} title={"Chat Logs"} />
			</Grid>
		</Grid>
	);
}
