import Grid from "@mui/material/Grid";
import Tooltip from "@mui/material/Tooltip";
import Typography from "@mui/material/Typography";
import { useQuery } from "@tanstack/react-query";
import { createFileRoute, stripSearchParams } from "@tanstack/react-router";
import {
	createMRTColumnHelper,
	type MRT_ColumnFiltersState,
	type MRT_PaginationState,
	type MRT_SortingState,
	useMaterialReactTable,
} from "material-react-table";
import { useCallback, useMemo } from "react";
import { z } from "zod/v4";
import { apiGetServerLogs, apiGetServers } from "../api/index.ts";
import { TextLink } from "../component/TextLink.tsx";
import {
	createDefaultTableOptions,
	filterValueNumberArray,
	makeRowActionsDefOptions,
	makeSchemaDefaults,
	makeSchemaState,
	type OnChangeFn,
	setColumnFilter,
} from "../component/table/options.ts";
import { SortableTable } from "../component/table/SortableTable.tsx";
import type { ServerLog, ServerSimple } from "../schema/server.ts";
import { stringToColour } from "../util/colours.ts";
import { renderDateTime } from "../util/time.ts";

const columnHelper = createMRTColumnHelper<ServerLog>();
const defaultOptions = createDefaultTableOptions<ServerLog>();
const defaultValues = { ...makeSchemaDefaults({ defaultColumn: "person_message_id" }), server_ids: [] };
const validateSearch = z
	.object({
		server_ids: z.number().array().optional().default([]),
	})
	.extend(makeSchemaState("person_message_id").shape);

export const Route = createFileRoute("/_admin/admin/serverlogs")({
	validateSearch,
	search: {
		middlewares: [stripSearchParams(defaultValues)],
	},
	head: ({ match }) => {
		return {
			meta: [{ name: "description", content: "Server Logs" }, match.context.title("Server Logs")],
		};
	},
	component: AdminServerlogs,
	loader: async ({ context }) => {
		const unsorted = await context.queryClient.ensureQueryData({
			queryKey: ["serversSimple"],
			queryFn: ({ signal }) => apiGetServers(signal) ?? [],
		});
		return (unsorted ?? []).sort((a, b) => {
			return a.server_name > b.server_name ? 1 : a.server_name < b.server_name ? -1 : 0;
		});
	},
});

function AdminServerlogs() {
	const search = Route.useSearch();
	const navigate = Route.useNavigate();
	const servers = Route.useLoaderData();

	const { data, isLoading, isError, isRefetching } = useQuery({
		queryKey: ["serverLogs", { search }],
		queryFn: async ({ signal }) => {
			const server_ids = filterValueNumberArray("server_ids", search.columnFilters);
			return (
				(await apiGetServerLogs(
					{
						server_ids,
					},
					signal,
				)) ?? []
			);
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

	const columns = useMemo(() => {
		return [
			columnHelper.accessor("server_id", {
				header: "Server ID",
				grow: false,
				enableSorting: false,
				filterVariant: "multi-select",
				filterSelectOptions: (servers ?? []).map((server: ServerSimple) => ({
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
							search={setColumnFilter(search, "server_ids", [cell.getValue()])}
							sx={{ color: stringToColour(row.original.server_name ?? "") }}
						>
							{row.original.server_name}
						</TextLink>
					</Tooltip>
				),
			}),

			columnHelper.accessor("server_name", {
				grow: false,
				enableSorting: false,
				enableColumnFilter: false,
				meta: {
					tooltip: "Short unique server identifier",
				},
				header: "Server Name",
				Cell: ({ cell, row }) => (
					<Typography sx={{ color: stringToColour(row.original.server_name) }}>{cell.getValue()}</Typography>
				),
			}),

			columnHelper.accessor("body", {
				header: "Log Message",
				enableSorting: false,
				grow: true,
			}),

			columnHelper.accessor("created_on", {
				header: "Created On",
				enableColumnFilter: false,
				grow: false,
				Cell: ({ cell }) => renderDateTime(cell.getValue()),
			}),
		];
	}, [servers, search]);

	const table = useMaterialReactTable({
		...defaultOptions,
		columns,
		data: data?.data ?? [],
		rowCount: data?.count ?? 0,
		enableFilters: true,
		displayColumnDefOptions: makeRowActionsDefOptions(2),
		manualFiltering: true,
		manualPagination: true,
		manualSorting: true,
		onColumnFiltersChange: setColumnFilters,
		onPaginationChange: setPagination,
		onSortingChange: setSorting,
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
			sorting: [{ id: "name", desc: false }],
			pagination: {
				pageIndex: 0,
				pageSize: 100,
			},
			columnVisibility: {
				server_id: true,
				server_name: true,
				body: true,
				created_on: true,
			},
		},
	});

	return (
		<Grid container spacing={2}>
			<Grid size={{ xs: 12 }}>
				<SortableTable table={table} title={"Server Logs"} />
			</Grid>
		</Grid>
	);
}
