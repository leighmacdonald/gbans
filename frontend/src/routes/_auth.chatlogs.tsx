import FlagIcon from "@mui/icons-material/Flag";
import RefreshIcon from "@mui/icons-material/Refresh";
import ReportIcon from "@mui/icons-material/Report";
import { IconButton, Typography } from "@mui/material";
import Grid from "@mui/material/Grid";
import Tooltip from "@mui/material/Tooltip";
import { useTheme } from "@mui/system";
import { createFileRoute, stripSearchParams, useNavigate } from "@tanstack/react-router";
import {
	createMRTColumnHelper,
	type MRT_ColumnFiltersState,
	type MRT_PaginationState,
	type MRT_SortingState,
	useMaterialReactTable,
} from "material-react-table";
import { useCallback, useMemo } from "react";
import z from "zod/v4";
import { IconButtonLink } from "../component/IconButtonLink.tsx";
import { PersonCell } from "../component/PersonCell.tsx";
import RouterLink from "../component/RouterLink.tsx";
import { RowActionContainer } from "../component/RowActionContainer.tsx";
import { TextLink } from "../component/TextLink.tsx";
import {
	createDefaultTableOptions,
	dateTimeColumnSize,
	filterValue,
	filterValueNumber,
	makeRowActionsDefOptions,
	makeSchemaDefaults,
	makeSchemaState,
	type OnChangeFn,
	setColumnFilter,
} from "../component/table/options.ts";
import { SortableTable } from "../component/table/SortableTable.tsx";
import { stringToColour } from "../util/colours.ts";
import { ensureFeatureEnabled } from "../util/features.ts";
import { renderDateTime } from "../util/time.ts";
import { useQuery, useSuspenseQuery } from "@connectrpc/connect-query";
import { servers } from "../rpc/servers/v1/servers-ServersService_connectquery.ts";
import { apiGetServers } from "../api";
import { query } from "../rpc/chat/v1/chat-ChatService_connectquery.ts";
import { keepPreviousData } from "@tanstack/react-query";
import type { Message } from "../rpc/chat/v1/chat_pb.ts";

const defaultValues = { ...makeSchemaDefaults({ defaultColumn: "person_message_id" }), flagged_only: false };
const validateSearch = z
	.object({
		flagged_only: z.boolean().optional().default(false),
	})
	.extend(makeSchemaState("person_message_id").shape);

export const Route = createFileRoute("/_auth/chatlogs")({
	component: ChatLogs,
	beforeLoad: ({ context }) => {
		ensureFeatureEnabled(context.appInfo.chatlogsEnabled);
	},
	validateSearch,
	search: {
		middlewares: [stripSearchParams(defaultValues)],
	},
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

const columnHelper = createMRTColumnHelper<Message>();
const defaultOptions = createDefaultTableOptions<Message>();

function ChatLogs() {
	const search = Route.useSearch();
	const { data: serverList } = useSuspenseQuery(servers);
	const navigate = useNavigate();
	const theme = useTheme();

	const setSorting: OnChangeFn<MRT_SortingState> = useCallback(
		async (updater) => {
			await navigate({
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
		async (updater) => {
			await navigate({
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
		async (updater) => {
			await navigate({
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
			columnHelper.accessor("serverId", {
				header: "Server",
				grow: false,
				enableSorting: false,
				filterVariant: "multi-select",
				filterSelectOptions: serverList.servers.map((server) => ({
					label: server.serverName,
					value: server.serverId,
				})),
				filterFn: (row, _, filterValue) => {
					return filterValue.length === 0 || filterValue.includes(row.original.serverId);
				},
				Cell: ({ row, cell }) => (
					<Tooltip title={row.original.serverName}>
						<TextLink
							to={"/chatlogs"}
							search={setColumnFilter(search, "server_id", [cell.getValue()])}
							sx={{ color: stringToColour(row.original.serverName ?? "") }}
						>
							{row.original.serverName}
						</TextLink>
					</Tooltip>
				),
			}),

			columnHelper.accessor("createdOn", {
				header: "Created",
				enableColumnFilter: false,
				grow: false,
				size: dateTimeColumnSize,
				Cell: ({ cell }) => renderDateTime(cell.getValue()),
			}),

			columnHelper.accessor("steamId", {
				header: "SteamID",
				grow: false,
				enableSorting: false,
				enableColumnFilter: true,
				filterFn: (row, _, filterValue) => {
					const query = filterValue.toLowerCase();
					if (query === "") {
						return true;
					}
					const value = row.original.steamId.toString();
					if (value.includes(query)) {
						return true;
					}
					return value.includes(query) || row.original.steamId === query;
				},
				Cell: ({ row }) => (
					<PersonCell
						steam_id={row.original.steamId}
						avatar_hash={row.original.avatarHash}
						personaname={row.original.personaName}
					>
						<RouterLink
							style={{ color: theme.palette.primary.light }}
							to={Route.fullPath}
							search={setColumnFilter(search, "steam_id", row.original.steamId)}
						>
							{row.original.personaName ?? row.original.steamId}
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
							color={row.original.autoFilterFlagged > 0 ? "error" : "inherit"}
						>
							{renderedCellValue}
						</Typography>
					);
				},
			}),
		];
	}, [servers, search, theme]);

	const sort = search.sorting?.find((sort) => sort);

	const { data, isLoading, isError, isRefetching, refetch } = useQuery(
		query,
		{
			serverId: filterValueNumber("server_id", search.columnFilters),
			steamId: BigInt(filterValue("steam_id", search.columnFilters)),
			query: filterValue("body", search.columnFilters),
			flaggedOnly: search.flagged_only,
			filter: {
				limit: BigInt(search.pagination?.pageSize ?? 25n),
				desc: sort ? sort.desc : true,
				offset: BigInt(search.pagination ? search.pagination.pageIndex * search.pagination.pageSize : 0),
				orderBy: sort ? sort.id : "created_on",
			},
		},
		{ placeholderData: keepPreviousData },
	);

	const table = useMaterialReactTable({
		...defaultOptions,
		columns,
		data: data ? data.messages : [],
		rowCount: Number(data ? data.count : 0),
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
				<Tooltip title={"Create Report"} key={row.original.personMessageId}>
					<IconButtonLink
						color={"error"}
						disabled={row.original.autoFilterFlagged > 0}
						to={"/report"}
						search={{
							person_message_id: Number(row.original.personMessageId),
							steam_id: String(row.original.steamId),
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
							onClick={async () => {
								await navigate({
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
