import { create } from "@bufbuild/protobuf";
import { useQuery } from "@connectrpc/connect-query";
import CloudDownload from "@mui/icons-material/CloudDownload";
import FlagIcon from "@mui/icons-material/Flag";
import RefreshIcon from "@mui/icons-material/Refresh";
import ReportIcon from "@mui/icons-material/Report";
import Grid from "@mui/material/Grid";
import IconButton from "@mui/material/IconButton";
import Link from "@mui/material/Link";
import Tooltip from "@mui/material/Tooltip";
import Typography from "@mui/material/Typography";
import { useTheme } from "@mui/system";
import { keepPreviousData } from "@tanstack/react-query";
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
	filterValueNumber,
	filterValueString,
	makeRowActionsDefOptions,
	makeSchemaDefaults,
	makeSchemaState,
	type OnChangeFn,
	setColumnFilter,
} from "../component/table/options.ts";
import { SortableTable } from "../component/table/SortableTable.tsx";
import { renderTableError } from "../error.tsx";
import { type Message, QueryRequestSchema } from "../rpc/chat/v1/chat_pb.ts";
import { query } from "../rpc/chat/v1/chat-ChatService_connectquery.ts";
import { servers } from "../rpc/servers/v1/servers-ServersService_connectquery.ts";
import { stringToColour } from "../util/colours.ts";
import { ensureFeatureEnabled } from "../util/features.ts";
import { renderTimestamp } from "../util/time.ts";
import { emptyOrNullString } from "../util/types.ts";

const defaultValues = { ...makeSchemaDefaults({ defaultColumn: "createdOn", defaultDesc: true }), flaggedOnly: false };
const validateSearch = z
	.object({
		flaggedOnly: z.boolean().optional().default(false),
	})
	.extend(makeSchemaState("createdOn").shape);

export const Route = createFileRoute("/_auth/chatlogs")({
	component: ChatLogs,
	beforeLoad: ({ context }) => {
		ensureFeatureEnabled(context.appInfo.chatlogsEnabled);
	},
	validateSearch,
	search: {
		middlewares: [stripSearchParams(defaultValues)],
	},
	head: ({ match }) => ({
		meta: [{ name: "description", content: "Browse in-game chat logs" }, match.context.title("Chat Logs")],
	}),
});

const columnHelper = createMRTColumnHelper<Message>();
const defaultOptions = createDefaultTableOptions<Message>();

function ChatLogs() {
	const search = Route.useSearch();
	const { data: serverList, isLoading: isLoadingServers } = useQuery(servers);
	const navigate = useNavigate();
	const theme = useTheme();

	const serversSorted = useMemo(() => {
		return serverList?.servers.toSorted((a, b) => a.serverId - b.serverId) ?? [];
	}, [serverList]);

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
				size: 100,
				enableSorting: false,
				filterVariant: "multi-select",
				filterSelectOptions: serversSorted.map((server) => ({
					label: server.serverName,
					value: server.serverId,
				})),
				Cell: ({ row, cell }) => (
					<Tooltip title={row.original.serverName}>
						<TextLink
							to={"/chatlogs"}
							search={setColumnFilter(search, "serverId", [cell.getValue()])}
							sx={{ color: stringToColour(row.original.serverName ?? "") }}
						>
							{serversSorted.find((s) => s.serverId === cell.getValue())?.serverName ?? ""}
						</TextLink>
					</Tooltip>
				),
			}),

			columnHelper.accessor("createdOn", {
				header: "Created",
				enableColumnFilter: false,
				grow: false,
				size: dateTimeColumnSize,
				Cell: ({ cell }) => renderTimestamp(cell.getValue()),
			}),

			columnHelper.accessor("demoId", {
				header: "Demo ID",
				enableColumnFilter: false,
				grow: false,
			}),

			columnHelper.accessor("demoTick", {
				header: "Demo Tick",
				enableColumnFilter: false,
				grow: false,
			}),

			columnHelper.accessor("steamId", {
				header: "SteamID",
				grow: true,
				size: 120,
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
						steamId={row.original.steamId}
						avatarHash={row.original.avatarHash}
						personaName={row.original.personaName}
					>
						<RouterLink
							style={{ color: theme.palette.primary.light }}
							to={Route.fullPath}
							search={setColumnFilter(search, "steamId", row.original.steamId)}
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
				enableColumnFilter: true,
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
	}, [search, theme, serversSorted]);

	const opts = useMemo(() => {
		const sort = search.sorting?.find((sort) => sort);

		const o = create(QueryRequestSchema, {
			filter: {
				limit: String(search.pagination?.pageSize ?? 25),
				desc: sort ? sort.desc : true,
				offset: String(search.pagination ? search.pagination.pageIndex * search.pagination.pageSize : 0),
				orderBy: sort ? sort.id : "createdOn",
			},
		});

		const serverId = filterValueNumber("serverId", search.columnFilters);
		if (serverId) {
			o.serverIds = [serverId];
		}
		try {
			const steamId = filterValueString("steamId", search.columnFilters);
			if (!emptyOrNullString(steamId)) {
				o.steamId = steamId;
			}
		} catch (e) {
			console.log(e);
		}
		o.flaggedOnly = search.flaggedOnly ?? undefined;
		const query = filterValueString("body", search.columnFilters);
		if (query) {
			o.query = query;
		}

		return o;
	}, [search]);

	const { data, isLoading, isError, isRefetching, refetch, error } = useQuery(query, opts, {
		placeholderData: keepPreviousData,
		retry: false,
	});

	const rowCount = useMemo(() => {
		const offset = (search.pagination?.pageIndex ?? 0) + 1;
		const rows = search.pagination?.pageSize ?? 25;
		return offset * rows + 1;
	}, [search]);

	const table = useMaterialReactTable({
		...defaultOptions,
		columns,
		data: data ? data.messages : [],
		rowCount,
		enableFilters: true,
		enableRowActions: true,
		autoResetPageIndex: true,
		displayColumnDefOptions: makeRowActionsDefOptions(3),
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
				serverId: true,
				sourceId: true,
				body: true,
				createdOn: true,
				demoTick: false,
				demoId: false,
			},
		},
		manualFiltering: true,
		manualPagination: true,
		manualSorting: true,
		onColumnFiltersChange: setColumnFilters,
		onPaginationChange: setPagination,
		onSortingChange: setSorting,
		muiToolbarAlertBannerProps: renderTableError(error),
		muiPaginationProps: {
			showLastButton: false,
		},
		renderRowActions: ({ row }) => (
			<RowActionContainer>
				<Tooltip title={"Create Report"} key={1}>
					<IconButtonLink
						color={"error"}
						disabled={row.original.autoFilterFlagged > 0}
						to={"/report"}
						search={{
							personMessageId: row.original.personMessageId,
							demoId: row.original.demoId,
							demoTick: row.original.demoTick,
							steamId: row.original.steamId,
						}}
					>
						<ReportIcon />
					</IconButtonLink>
				</Tooltip>
				{!emptyOrNullString(row.original.matchId) && (
					<Tooltip title={"Match Results"} key={1}>
						<IconButtonLink
							color={"error"}
							disabled={row.original.autoFilterFlagged > 0}
							to={"/match/$matchId"}
							params={{
								matchId: row.original.matchId,
							}}
						>
							<ReportIcon />
						</IconButtonLink>
					</Tooltip>
				)}
				{row.original.assestId && (
					<Tooltip title={"Download Demo"} key={2}>
						<IconButton
							component={Link}
							key={"dl-link"}
							color={"success"}
							href={`/asset/${row.original.assestId}`}
							disabled={row.original.autoFilterFlagged > 0}
						>
							<CloudDownload />
						</IconButton>
					</Tooltip>
				)}
			</RowActionContainer>
		),
	});

	if (isLoadingServers) {
		return;
	}

	return (
		<Grid container spacing={2}>
			<Grid size={{ xs: 12 }}>
				<SortableTable
					unknownRowCount={true}
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
										flaggedOnly: !search.flaggedOnly,
									},
								});
							}}
						>
							<IconButton sx={{ color: search.flaggedOnly ? "error" : "primary.contrastText" }}>
								<FlagIcon />
							</IconButton>
						</Tooltip>,
					]}
				/>
			</Grid>
		</Grid>
	);
}
