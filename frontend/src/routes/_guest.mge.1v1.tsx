/** biome-ignore-all lint/correctness/noChildrenProp: ts-form made me do it! */
import Grid from "@mui/material/Grid";
import Typography from "@mui/material/Typography";
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
import { PersonCell } from "../component/PersonCell.tsx";
import RouterLink from "../component/RouterLink.tsx";
import {
	createDefaultTableOptions,
	makeRowActionsDefOptions,
	makeSchemaDefaults,
	makeSchemaState,
	type OnChangeFn,
	setColumnFilter,
} from "../component/table/options.ts";
import { SortableTable } from "../component/table/SortableTable.tsx";
import { renderDateTime } from "../util/time.ts";
import { getHistory } from "../rpc/mge/v1/mge-MGEService_connectquery.ts";
import { useQuery } from "@connectrpc/connect-query";
import type { Duel } from "../rpc/mge/v1/mge_pb.ts";

const columnHelper = createMRTColumnHelper<Duel>();
const defaultOptions = createDefaultTableOptions<Duel>();
const defaultValues = makeSchemaDefaults({ defaultColumn: "rating" });
const validateSearch = makeSchemaState("rating");

export const Route = createFileRoute("/_guest/mge/1v1")({
	component: MGEOverall,
	validateSearch,
	search: {
		middlewares: [stripSearchParams(defaultValues)],
	},
	head: ({ match }) => ({
		meta: [
			{
				name: "description",
				content: "MGE 1v1 Match History",
			},
			match.context.title("MGE 1v1 Match History"),
		],
	}),
});

function MGEOverall() {
	const navigate = useNavigate();
	const search = Route.useSearch();
	const theme = useTheme();

	const { data, isLoading, isError, isRefetching } = useQuery(getHistory);

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
			columnHelper.accessor("duelId", {
				header: "ID",
				grow: false,
				Cell: ({ cell }) => <Typography>{cell.getValue()}</Typography>,
			}),
			columnHelper.accessor("winner", {
				grow: true,
				header: "Winner",
				Cell: ({ row }) => (
					<PersonCell
						steam_id={row.original.winner}
						avatar_hash={row.original.winnerAvatarHash}
						personaname={row.original.winnerPersonaName}
					>
						<RouterLink
							style={{
								color:
									theme.palette.mode === "dark"
										? theme.palette.primary.light
										: theme.palette.primary.dark,
							}}
							to={Route.fullPath}
							search={setColumnFilter(search, "winner", row.original.winner)}
						>
							{row.original.winnerPersonaName}
						</RouterLink>
					</PersonCell>
				),
			}),
			columnHelper.accessor("loser", {
				grow: true,
				header: "Loser",
				Cell: ({ row }) => (
					<PersonCell
						steam_id={row.original.loser}
						avatar_hash={row.original.loserAvatarHash}
						personaname={row.original.loserPersonaName}
					>
						<RouterLink
							style={{
								color:
									theme.palette.mode === "dark"
										? theme.palette.primary.light
										: theme.palette.primary.dark,
							}}
							to={Route.fullPath}
							search={setColumnFilter(search, "loser", row.original.loser)}
						>
							{row.original.loserPersonaName}
						</RouterLink>
					</PersonCell>
				),
			}),
			columnHelper.accessor("winnerScore", {
				enableColumnFilter: false,
				grow: false,
				header: "W.Score",
			}),
			columnHelper.accessor("loserScore", {
				enableColumnFilter: false,
				grow: false,
				header: "L.Score",
			}),
			columnHelper.accessor("winLimit", {
				enableColumnFilter: false,
				grow: false,
				header: "Winlimit",
			}),
			columnHelper.accessor("gameTime", {
				enableColumnFilter: false,
				grow: false,
				header: "Game Time",
				Cell: ({ row }) => renderDateTime(row.original.gameTime),
			}),
			columnHelper.accessor("mapName", {
				enableColumnFilter: false,
				grow: false,
				header: "Map Name",
			}),
			columnHelper.accessor("arenaName", {
				enableColumnFilter: false,
				grow: true,
				header: "Arena Name",
			}),
		],
		[search, theme],
	);

	const table = useMaterialReactTable({
		...defaultOptions,
		columns,
		data: data?.history ?? [],
		rowCount: Number(data?.count ?? 0),
		enableFilters: true,
		enableHiding: true,
		enableFacetedValues: true,
		onColumnFiltersChange: setColumnFilters,
		onPaginationChange: setPagination,
		onSortingChange: setSorting,
		manualFiltering: true,
		manualPagination: true,
		manualSorting: true,
		displayColumnDefOptions: makeRowActionsDefOptions(2),
		state: {
			isLoading: isLoading || isRefetching,
			showProgressBars: isRefetching,
			showAlertBanner: isError,
			columnFilters: search.columnFilters,
			pagination: search.pagination,
			sorting: search.sorting,
		},
		initialState: {
			...defaultOptions.initialState,
			columnVisibility: {
				duel_id: false,
				winlimit: false,
			},
		},
	});
	return (
		<Grid container>
			<Grid size={{ xs: 12 }}>
				<SortableTable table={table} title={"MGE 1v1 Match History"} />
			</Grid>
		</Grid>
	);
}
