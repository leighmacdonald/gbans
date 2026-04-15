/** biome-ignore-all lint/correctness/noChildrenProp: ts-form made me do it! */

import Filter1Icon from "@mui/icons-material/Filter1";
import Filter2Icon from "@mui/icons-material/Filter2";
import Grid from "@mui/material/Grid";
import Tooltip from "@mui/material/Tooltip";
import Typography from "@mui/material/Typography";
import { useQuery } from "@tanstack/react-query";
import { createFileRoute, stripSearchParams, useNavigate } from "@tanstack/react-router";
import {
	createMRTColumnHelper,
	type MRT_ColumnFiltersState,
	type MRT_PaginationState,
	type MRT_SortingState,
	useMaterialReactTable,
} from "material-react-table";
import { useCallback, useMemo } from "react";
import { apiMGEOverall } from "../api/mge.ts";
import { IconButtonLink } from "../component/IconButtonLink.tsx";
import { PersonCell } from "../component/PersonCell.tsx";
import { RowActionContainer } from "../component/RowActionContainer.tsx";
import {
	createDefaultTableOptions,
	filterValue,
	makeRowActionsDefOptions,
	makeSchemaDefaults,
	makeSchemaState,
	type OnChangeFn,
	setColumnFilter,
} from "../component/table/options.ts";
import { SortableTable } from "../component/table/SortableTable.tsx";
import type { MGEStat } from "../schema/mge.ts";
import { ensureFeatureEnabled } from "../util/features.ts";
import { renderDate } from "../util/time.ts";

const columnHelper = createMRTColumnHelper<MGEStat>();
const defaultOptions = createDefaultTableOptions<MGEStat>();
const defaultValues = makeSchemaDefaults({ defaultColumn: "rating" });
const validateSearch = makeSchemaState("rating");

export const Route = createFileRoute("/_guest/mge/")({
	component: MGEOverall,
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
				content: "MGE Standings",
			},
			match.context.title("MGE Standings"),
		],
	}),
});

function MGEOverall() {
	const navigate = useNavigate();
	const search = Route.useSearch();

	const { data, isLoading, isError, isRefetching } = useQuery({
		queryKey: ["mgeStats", { search }],
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
	});

	const setSorting: OnChangeFn<MRT_SortingState> = useCallback(
		async (updater) => {
			await navigate({
				to: "/mge/1v1",
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
				to: "/mge",
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
				to: "/mge",
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
			columnHelper.accessor("rating", {
				header: "Rating",
				grow: false,
				Cell: ({ cell }) => <Typography>{cell.getValue()}</Typography>,
			}),
			columnHelper.accessor("steam_id", {
				grow: true,
				header: "Player",
				Cell: ({ row }) => (
					<PersonCell
						steam_id={row.original.steam_id}
						avatar_hash={row.original.avatarhash}
						personaname={row.original.personaname}
					/>
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
			columnHelper.accessor("last_played", {
				header: "Last Played",
				enableColumnFilter: false,
				enableSorting: false,
				grow: false,
				Cell: ({ cell }) => renderDate(cell.getValue()),
			}),
		];
	}, []);

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
		enableRowNumbers: true,
		displayColumnDefOptions: makeRowActionsDefOptions(2),
		state: {
			isLoading: isLoading || isRefetching,
			showProgressBars: isRefetching,
			showAlertBanner: isError,
			columnFilters: search.columnFilters,
			pagination: search.pagination,
			sorting: search.sorting,
		},
		enableRowActions: true,
		renderRowActions: ({ row }) => (
			<RowActionContainer>
				<Tooltip title={"1v1 History"} key={1}>
					<IconButtonLink
						color="primary"
						to={"/mge/1v1"}
						search={setColumnFilter(search, "winner", row.original.steam_id)}
					>
						<Filter1Icon />
					</IconButtonLink>
				</Tooltip>
				<Tooltip title={"2v2 History"} key={2}>
					<IconButtonLink
						color="secondary"
						to={"/mge/2v2"}
						search={setColumnFilter(search, "winner", row.original.steam_id)}
					>
						<Filter2Icon />
					</IconButtonLink>
				</Tooltip>
			</RowActionContainer>
		),
		initialState: {
			...defaultOptions.initialState,
			columnVisibility: {
				demo_id: false,
				server_id: true,
				created_on: true,
			},
		},
	});
	return (
		<Grid container>
			<Grid size={{ xs: 12 }}>
				<SortableTable table={table} title={"MGE Overall Rankings"} />
			</Grid>
		</Grid>
	);
}
