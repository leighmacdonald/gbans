import FlagIcon from "@mui/icons-material/Flag";
import RefreshIcon from "@mui/icons-material/Refresh";
import ReportIcon from "@mui/icons-material/Report";
import { IconButton, Typography } from "@mui/material";
import Grid from "@mui/material/Grid";
import Tooltip from "@mui/material/Tooltip";
import { useTheme } from "@mui/system";
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
import RouterLink from "../component/RouterLink.tsx";
import { RowActionContainer } from "../component/RowActionContainer.tsx";
import { TextLink } from "../component/TextLink.tsx";
import {
	createDefaultTableOptions,
	dateTimeColumnSize,
	filterValue,
	makeRowActionsDefOptions,
	makeSchemaState,
	type OnChangeFn,
	setColumnFilter,
} from "../component/table/options.ts";
import { SortableTable } from "../component/table/SortableTable.tsx";
import type { PersonMessage } from "../schema/people.ts";
import { stringToColour } from "../util/colours.ts";
import { ensureFeatureEnabled } from "../util/features.ts";
import { renderDateTime } from "../util/time.ts";

const validateSearch = z
	.object({
		flagged_only: z.boolean().optional().default(false),
	})
	.extend(makeSchemaState("person_message_id").shape);

export const Route = createFileRoute("/_auth/chatlogs")({
	component: ChatLogs,
	beforeLoad: ({ context }) => {
		ensureFeatureEnabled(context.appInfo.chatlogs_enabled);
	},
	validateSearch,
	loader: async ({ context }) => {
		const unsorted = await context.queryClient.ensureQueryData({
			queryKey: ["serversSimple"],
			queryFn: ({ signal }) => apiGetServers(signal),
		});
		return unsorted.sort((a, b) => {
			return a.server_name > b.server_name ? 1 : a.server_name < b.server_name ? -1 : 0;
		});
	},
	head: ({ match }) => ({
		meta: [{ name: "description", content: "Browse in-game chat logs" }, match.context.title("Chat Logs")],
	}),
});

const columnHelper = createMRTColumnHelper<PersonMessage>();
const defaultOptions = createDefaultTableOptions<PersonMessage>();

function ChatLogs() {
	const search = Route.useSearch();
	const servers = Route.useLoaderData();
	const navigate = useNavigate();
	const theme = useTheme();

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
							to={"/chatlogs"}
							search={setColumnFilter(search, "server_id", [cell.getValue()])}
							sx={{ color: stringToColour(row.original.server_name ?? "") }}
						>
							{row.original.server_name}
						</TextLink>
					</Tooltip>
				),
			}),

			columnHelper.accessor("created_on", {
				header: "Created",
				enableColumnFilter: false,
				grow: false,
				size: dateTimeColumnSize,
				Cell: ({ cell }) => renderDateTime(cell.getValue()),
			}),

			columnHelper.accessor("steam_id", {
				header: "SteamID",
				grow: false,
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
						steam_id={row.original.steam_id}
						avatar_hash={row.original.avatar_hash}
						personaname={row.original.persona_name}
					>
						<RouterLink
							style={{ color: theme.palette.primary.light }}
							to={Route.fullPath}
							search={setColumnFilter(search, "steam_id", row.original.steam_id)}
						>
							{row.original.persona_name ?? row.original.steam_id}
						</RouterLink>
					</PersonCell>
				),
			}),

			columnHelper.accessor("body", {
				header: "Message",
				grow: true,
				enableSorting: false,
				Cell: ({ row, renderedCellValue }) => {
					return (
						<Typography
							padding={0}
							variant={"body1"}
							color={row.original.auto_filter_flagged > 0 ? "error" : "inherit"}
						>
							{renderedCellValue}
						</Typography>
					);
				},
			}),
		];
	}, [servers, search, theme]);

	const { data, isLoading, isError, isRefetching, refetch } = useQuery({
		queryKey: ["chatlogs", { search }],
		queryFn: async ({ signal }) => {
			const server_id = filterValue("server_id", search.columnFilters);
			const steam_id = filterValue("steam_id", search.columnFilters);
			const body = filterValue("body", search.columnFilters);
			const sort = search.sorting?.find((sort) => sort);

			return await apiGetMessages(
				{
					server_id: server_id ? Number(server_id) : 0,
					personaname: "",
					query: body ? String(body) : "",
					source_id: steam_id ? String(steam_id) : "",
					limit: search.pagination?.pageSize,
					offset: search.pagination ? search.pagination.pageIndex * search.pagination.pageSize : undefined,
					order_by: sort ? sort.id : "created_on",
					desc: sort ? sort.desc : true,
					flagged_only: search.flagged_only,
				},
				signal,
			);
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
		displayColumnDefOptions: makeRowActionsDefOptions(1),
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
		renderRowActions: ({ row }) => (
			<RowActionContainer>
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
				</Tooltip>
			</RowActionContainer>
		),
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
							<IconButton sx={{ color: search.flagged_only ? "error" : "primary.contrastText" }}>
								<FlagIcon />
							</IconButton>
						</Tooltip>,
					]}
				/>
			</Grid>
		</Grid>
	);
}
