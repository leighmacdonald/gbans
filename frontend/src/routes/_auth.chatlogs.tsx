import FlagIcon from "@mui/icons-material/Flag";
import RefreshIcon from "@mui/icons-material/Refresh";
import ReportIcon from "@mui/icons-material/Report";
import { IconButton, TableCell, Typography, useTheme } from "@mui/material";
import Grid from "@mui/material/Grid";
import Tooltip from "@mui/material/Tooltip";
import { keepPreviousData, useQuery } from "@tanstack/react-query";
import { createFileRoute, useNavigate } from "@tanstack/react-router";
import {
	createMRTColumnHelper,
	type MRT_ColumnFiltersState,
	type MRT_PaginationState,
	type MRT_SortingState,
	useMaterialReactTable,
} from "material-react-table";
import { useCallback, useMemo } from "react";
import z from "zod/v4";
import { apiGetMessages, apiGetServers } from "../api";
import { IconButtonLink } from "../component/IconButtonLink.tsx";
import { PersonCell } from "../component/PersonCell.tsx";
import { createDefaultTableOptions, makeSchemaState, type OnChangeFn } from "../component/table/options.ts";
import { SortableTable } from "../component/table/SortableTable.tsx";
import { TableCellRelativeDateField } from "../component/table/TableCellRelativeDateField.tsx";
import type { PersonMessage } from "../schema/people.ts";
import { stringToColour } from "../util/colours.ts";
import { ensureFeatureEnabled } from "../util/features.ts";

const schema = z
	.object({
		flagged_only: z.boolean().catch(false),
	})
	.extend(makeSchemaState({ defaultSortColumn: "person_message_id" }));

export const Route = createFileRoute("/_auth/chatlogs")({
	component: ChatLogs,
	beforeLoad: ({ context }) => {
		ensureFeatureEnabled(context.appInfo.chatlogs_enabled);
	},
	validateSearch: schema,
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
	const search = Route.useSearch();
	const { servers } = Route.useLoaderData();
	const navigate = useNavigate();
	const theme = useTheme();

	const setSorting: OnChangeFn<MRT_SortingState> = useCallback(
		(updater) => {
			navigate({
				to: Route.fullPath,
				search: {
					...search,
					sorting: typeof updater === "function" ? updater(search.sorting) : updater,
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
					columnFilters: typeof updater === "function" ? updater(search.columnFilters) : updater,
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
					pagination: typeof updater === "function" ? updater(search.pagination) : updater,
				},
			});
		},
		[search, navigate],
	);

	const columns = useMemo(() => {
		return [
			columnHelper.accessor("server_id", {
				header: "Srv",
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
				Cell: (info) => <TableCellRelativeDateField date={info.row.original.created_on} suffix />,
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
				Cell: ({ cell, row }) => {
					return (
						<Typography
							padding={0}
							variant={"body1"}
							color={row.original.auto_filter_flagged > 0 ? "error" : "inherit"}
						>
							{cell.getValue()}
						</Typography>
					);
				},
			}),
		];
	}, [theme.palette.mode, servers]);

	const { data, isLoading, isError, isRefetching, refetch } = useQuery({
		queryKey: ["chatlogs", { search }],
		queryFn: async () => {
			const server_id = search.columnFilters.find((filter) => filter.id === "server_id")?.value;
			const steam_id = search.columnFilters.find((filter) => filter.id === "steam_id")?.value;
			const body = search.columnFilters.find((filter) => filter.id === "body")?.value;
			const sort = search.sorting.find((sort) => sort);

			return await apiGetMessages({
				server_id: server_id ? Number(server_id) : 0,
				personaname: "",
				query: body ? String(body) : "",
				source_id: steam_id ? String(steam_id) : "",
				limit: search.pagination.pageSize,
				offset: search.pagination.pageIndex * search.pagination.pageSize,
				order_by: sort ? sort.id : "created_on",
				desc: sort ? sort.desc : true,
				flagged_only: search.flagged_only,
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
		enableRowActions: true,
		state: {
			columnFilters: search.columnFilters,
			isLoading: isLoading || isRefetching,
			pagination: search.pagination,
			showAlertBanner: isError,
			showProgressBars: isRefetching,
			sorting: search.sorting,
		},
		initialState: {
			...defaultOptions.initialState,
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
		onColumnFiltersChange: setColumnFilters,
		onPaginationChange: setPagination,
		onSortingChange: setSorting,
		renderRowActionMenuItems: ({ row }) => [
			<Tooltip title={"Create Report"} key={row.original.person_message_id}>
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
			</Tooltip>,
		],
	});

	return (
		<Grid container spacing={2}>
			<Grid size={{ xs: 12 }}>
				<SortableTable
					table={table}
					title={"Chat Logs"}
					buttons={[
						<Tooltip arrow title="Refresh Data" key="refresh">
							<IconButton onClick={() => refetch()} sx={{ color: "primary.contrastText" }}>
								<RefreshIcon />
							</IconButton>
						</Tooltip>,
						<Tooltip
							arrow
							title="Flagged Only"
							key="flagged"
							onClick={() => {
								navigate({
									to: Route.fullPath,
									search: {
										...search,
										flagged_only: !search.flagged_only,
									},
								});
							}}
						>
							<IconButton color={search.flagged_only ? "error" : "success"}>
								<FlagIcon />
							</IconButton>
						</Tooltip>,
					]}
				/>
			</Grid>
		</Grid>
	);
}
