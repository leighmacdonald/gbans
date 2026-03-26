/** biome-ignore-all lint/correctness/noChildrenProp: ts-form made me do it! */
import Grid from "@mui/material/Grid";
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
import { apiGetServers } from "../api/index.ts";
import {
	createDefaultTableOptions,
	filterValue,
	makeSchemaState,
	type OnChangeFn,
	setColumnFilter,
} from "../component/table/options.ts";
import { SortableTable } from "../component/table/SortableTable.tsx";
import { ensureFeatureEnabled } from "../util/features.ts";
import { MGEStat } from "../schema/mge.ts";
import { apiMGEOverall } from "../api/mge.ts";
import { PersonCell } from "../component/PersonCell.tsx";
import RouterLink from "../component/RouterLink.tsx";
import { useTheme } from "@mui/system";
import { renderDate } from "../util/time.ts";

const columnHelper = createMRTColumnHelper<MGEStat>();
const defaultOptions = createDefaultTableOptions<MGEStat>();
const validateSearch = makeSchemaState("rating");

export const Route = createFileRoute("/_guest/mge")({
	component: MGEOverall,
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
		})

    return unsorted.sort((a, b) =>
      a.server_name > b.server_name ? 1 : a.server_name < b.server_name ? -1 : 0);
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

function MGEOverall() {
	const servers = Route.useLoaderData();
	const navigate = useNavigate();
	const search = Route.useSearch();
  const theme = useTheme();

	const { data, isLoading, isError, isRefetching } = useQuery({
		queryKey: ["mgeStats", {search}],
    queryFn: async ({ signal }) => {
      const sort = search.sorting?.find((sort) => sort);
      const steam_id = filterValue("steam_id", search.columnFilters);

      return await apiMGEOverall(signal, {
       	limit: search.pagination?.pageSize,
				offset: search.pagination ? search.pagination.pageIndex * search.pagination.pageSize : undefined,
				order_by: sort ? sort.id : "created_on",
        desc: sort ? sort.desc : true,
				steam_id: steam_id ?? undefined,
			});
		},
	})

	const setSorting: OnChangeFn<MRT_SortingState> = useCallback(
		(updater) => {
			navigate({
				to: Route.fullPath,
				search: {
					...search,
					sorting: typeof updater === "function" ? updater(search.sorting ?? []) : updater,
				},
			})
		},
		[search, navigate],
	)

	const setColumnFilters: OnChangeFn<MRT_ColumnFiltersState> = useCallback(
		(updater) => {
			navigate({
				to: Route.fullPath,
				search: {
					...search,
					columnFilters: typeof updater === "function" ? updater(search.columnFilters ?? []) : updater,
				},
			})
		},
		[search, navigate],
	)

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
			})
		},
		[search, navigate],
	)
	const columns = useMemo(() => {
		return [
			columnHelper.accessor("rating", {
				header: "Rating",
				grow: false,
				Cell: ({ cell }) => <Typography>{cell.getValue()}</Typography>,
			}),
			columnHelper.accessor("steam_id", {
				grow: true,
				header: "Player",
				Cell: ({ row }) => (
          <PersonCell steam_id={row.original.steam_id} avatar_hash={row.original.avatarhash} personaname={row.original.personaname} >
            <RouterLink
							style={{ color: theme.palette.mode === "light" ? theme.palette.primary.light : theme.palette.secondary.light }}
							to={Route.fullPath}
							search={setColumnFilter(search, "steam_id", row.original.steam_id)}
						>
							{row.original.personaname ?? row.original.steam_id}
						</RouterLink>
					</PersonCell>
				),
			}),
      columnHelper.accessor("wins", {
        enableColumnFilter: false,
        grow: false,
				header: "Wins",

			}),
			columnHelper.accessor("losses", {
        enableColumnFilter: false,
				grow: false,
				header: "Loses",
			}),
			columnHelper.accessor("lastplayed", {
				header: "Last Played",
				enableColumnFilter: false,
				enableSorting: false,
				grow: false,
				Cell: ({ cell }) => renderDate(cell.getValue()),
			}),
		]
	}, [servers, search, theme]);

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
				demo_id: false,
				server_id: true,
				created_on: true,
			},
		},
	})
	return (
		<Grid container>
			<Grid size={{ xs: 12 }}>
				<SortableTable table={table} title={"MGE Overall Rankings"} />
			</Grid>
		</Grid>
	)
}
