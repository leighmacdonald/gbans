import { type Timestamp, timestampDate } from "@bufbuild/protobuf/wkt";
import { useQuery } from "@connectrpc/connect-query";
import { Tooltip, Typography } from "@mui/material";
import Grid from "@mui/material/Grid";
import { createFileRoute, stripSearchParams, useNavigate } from "@tanstack/react-router";
import { formatDistanceToNowStrict } from "date-fns/formatDistanceToNowStrict";
import {
	createMRTColumnHelper,
	type MRT_ColumnFiltersState,
	type MRT_PaginationState,
	type MRT_SortingState,
	useMaterialReactTable,
} from "material-react-table";
import { useCallback, useMemo } from "react";
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
import { renderTableError } from "../error.tsx";
import { servers } from "../rpc/servers/v1/servers-ServersService_connectquery.ts";
import type { PlayerMatchHistory } from "../rpc/stats/v1/stats_pb.ts";
import { matchesWithPlayer } from "../rpc/stats/v1/stats-StatsService_connectquery.ts";
import { stringToColour } from "../util/colours.ts";
import { renderTimestamp } from "../util/time.ts";

const validateSearch = makeSchemaState("matchId");
const columnHelper = createMRTColumnHelper<PlayerMatchHistory>();
const defaultOptions = createDefaultTableOptions<PlayerMatchHistory>();
const defaultValues = makeSchemaDefaults({ defaultColumn: "createdOn" });

export const Route = createFileRoute("/_auth/matches/$steamId")({
	component: ProfileMatchesPage,
	validateSearch,
	search: {
		middlewares: [stripSearchParams(defaultValues)],
	},
	head: () => ({
		meta: [{ name: "description", content: "Player Match History" }],
	}),
});

function ProfileMatchesPage() {
	const { steamId } = Route.useParams();
	const { data: serverList, isLoading: isLoadingServers } = useQuery(servers);
	const { data, isLoading, isError, error } = useQuery(
		matchesWithPlayer,
		{ steamId },
		{ enabled: !isLoadingServers },
	);

	const search = Route.useSearch();
	const navigate = useNavigate();

	const matches = useMemo(() => {
		const matchList = data?.matches ?? [];
		console.log(matchList);
		return matchList;
	}, [data]);

	const setSorting: OnChangeFn<MRT_SortingState> = useCallback(
		(updater) => {
			navigate({
				to: Route.fullPath,
				params: {
					steamId,
				},
				search: {
					...search,
					sorting: typeof updater === "function" ? updater(search.sorting ?? []) : updater,
				},
			});
		},
		[search, navigate, steamId],
	);

	const setColumnFilters: OnChangeFn<MRT_ColumnFiltersState> = useCallback(
		(updater) => {
			navigate({
				to: Route.fullPath,
				params: {
					steamId,
				},
				search: {
					...search,
					columnFilters: typeof updater === "function" ? updater(search.columnFilters ?? []) : updater,
				},
			});
		},
		[search, navigate, steamId],
	);

	const setPagination: OnChangeFn<MRT_PaginationState> = useCallback(
		(updater) => {
			navigate({
				to: Route.fullPath,
				params: {
					steamId,
				},
				search: {
					...search,
					pagination: search.pagination
						? typeof updater === "function"
							? updater(search.pagination)
							: updater
						: undefined,
				},
			});
		},
		[search, navigate, steamId],
	);
	const columns = useMemo(
		() => [
			columnHelper.accessor("bucketName", {
				grow: false,
				header: "Bucket",
				// Cell: ({ cell }) => (
				// 	<TextLink to={`/ban/$banId`} params={{ banId: String(cell.getValue()) }}>
				// 		{`#${cell.getValue()}`}
				// 	</TextLink>
				// ),
			}),
			columnHelper.accessor("createdOn", {
				header: "Created On",
				grow: false,
				Cell: ({ cell }) => (
					<Tooltip
						title={formatDistanceToNowStrict(timestampDate(cell.getValue() as Timestamp), {
							addSuffix: true,
						})}
					>
						<Typography>{renderTimestamp(cell.getValue())}</Typography>
					</Tooltip>
				),
			}),

			columnHelper.accessor("serverId", {
				header: "Server",
				grow: true,
				enableSorting: false,
				filterVariant: "multi-select",
				filterSelectOptions: serverList?.servers.map((server) => ({
					label: server.serverName,
					value: server.serverId,
				})),
				filterFn: (row, _, filterValue) => {
					return filterValue.length === 0 || filterValue.includes(row.original.serverId);
				},
				Cell: ({ row, cell }) => (
					<Tooltip title={row.original.serverName}>
						<TextLink
							to={"/matches/$steamId"}
							params={{ steamId }}
							search={setColumnFilter(search, "serverId", [cell.getValue()])}
							sx={{ color: stringToColour(row.original.serverName ?? "") }}
						>
							{row.original.serverName}
						</TextLink>
					</Tooltip>
				),
			}),

			columnHelper.accessor("mapName", {
				enableColumnFilter: true,
				enableSorting: false,
				grow: true,
				header: "Map",
			}),
		],
		[search, steamId, serverList],
	);

	const table = useMaterialReactTable({
		...defaultOptions,
		columns,
		data: matches,
		enableFilters: true,
		enableFacetedValues: true,
		onColumnFiltersChange: setColumnFilters,
		onPaginationChange: setPagination,
		onSortingChange: setSorting,
		displayColumnDefOptions: makeRowActionsDefOptions(2),
		state: {
			isLoading,
			showAlertBanner: isError,
			columnFilters: search.columnFilters,
			sorting: search.sorting,
			pagination: search.pagination,
		},
		initialState: {
			...defaultOptions.initialState,
			columnVisibility: {
				sourceId: false,
				targetId: true,
				reason: true,
				evadeOk: false,
				deleted: false,
				validUntil: true,
				createdOn: true,
				active: false,
				reportId: true,
				cidr: false,
			},
		},
		muiToolbarAlertBannerProps: renderTableError(error),
		enableRowActions: true,
	});
	return (
		<Grid container spacing={2}>
			<Grid size={{ xs: 12, md: 12 }}>
				<SortableTable table={table} title={"Match History"} />
			</Grid>
		</Grid>
	);
}
