/** biome-ignore-all lint/correctness/noChildrenProp: ts-form made me do it! */
import Grid from "@mui/material/Grid";
import Typography from "@mui/material/Typography";
import { useTheme } from "@mui/system";
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
import { apiMGEHistory } from "../api/mge.ts";
import { PersonCell } from "../component/PersonCell.tsx";
import RouterLink from "../component/RouterLink.tsx";
import {
	createDefaultTableOptions,
	filterValue,
	makeRowActionsDefOptions,
	makeSchemaState,
	type OnChangeFn,
	setColumnFilter,
} from "../component/table/options.ts";
import { SortableTable } from "../component/table/SortableTable.tsx";
import { DuelMode, type MGEHistory } from "../schema/mge.ts";
import { renderDateTime } from "../util/time.ts";

const columnHelper = createMRTColumnHelper<MGEHistory>();
const defaultOptions = createDefaultTableOptions<MGEHistory>();
const validateSearch = makeSchemaState("game_time");

export const Route = createFileRoute("/_guest/mge/1v1")({
	component: MGEOverall,
	validateSearch,
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

	const { data, isLoading, isError, isRefetching } = useQuery({
		queryKey: ["mgeHistory", { search }],
		queryFn: async ({ signal }) => {
			const sort = search.sorting?.find((sort) => sort);
			const winner = filterValue("winner", search.columnFilters);
			const loser = filterValue("loser", search.columnFilters);

			return await apiMGEHistory(signal, {
				limit: search.pagination?.pageSize,
				offset: search.pagination ? search.pagination.pageIndex * search.pagination.pageSize : undefined,
				order_by: sort ? sort.id : "game_time",
				desc: sort ? sort.desc : true,
				winner,
				loser,
				mode: DuelMode.OneVsOne,
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
			columnHelper.accessor("duel_id", {
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
						avatar_hash={row.original.winner_avatarhash}
						personaname={row.original.winner_personaname}
					>
						<RouterLink
							style={{
								color:
									theme.palette.mode === "light"
										? theme.palette.primary.light
										: theme.palette.secondary.light,
							}}
							to={Route.fullPath}
							search={setColumnFilter(search, "winner", row.original.winner)}
						>
							{row.original.winner_personaname}
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
						avatar_hash={row.original.loser_avatarhash}
						personaname={row.original.loser_personaname}
					>
						<RouterLink
							style={{
								color:
									theme.palette.mode === "light"
										? theme.palette.primary.light
										: theme.palette.secondary.light,
							}}
							to={Route.fullPath}
							search={setColumnFilter(search, "loser", row.original.loser)}
						>
							{row.original.loser_personaname}
						</RouterLink>
					</PersonCell>
				),
			}),
			columnHelper.accessor("winner_score", {
				enableColumnFilter: false,
				grow: false,
				header: "W.Score",
			}),
			columnHelper.accessor("loser_score", {
				enableColumnFilter: false,
				grow: false,
				header: "L.Score",
			}),
			columnHelper.accessor("winlimit", {
				enableColumnFilter: false,
				grow: false,
				header: "Winlimit",
			}),
			columnHelper.accessor("game_time", {
				enableColumnFilter: false,
				grow: false,
				header: "Game Time",
				Cell: ({ row }) => renderDateTime(row.original.game_time),
			}),
			columnHelper.accessor("map_name", {
				enableColumnFilter: false,
				grow: false,
				header: "Map Name",
			}),
			columnHelper.accessor("arena_name", {
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
		data: data?.data ?? [],
		rowCount: data?.count ?? 0,
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
