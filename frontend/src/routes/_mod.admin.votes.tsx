/** biome-ignore-all lint/correctness/noChildrenProp: form needs it */

import { useTheme } from "@mui/material";
import Grid from "@mui/material/Grid";
import Tooltip from "@mui/material/Tooltip";
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
import { apiGetServers } from "../api/server.ts";
import { apiVotesQuery } from "../api/votes.ts";
import { PersonCell } from "../component/PersonCell.tsx";
import RouterLink from "../component/RouterLink.tsx";
import { TextLink } from "../component/TextLink.tsx";
import { BoolCell } from "../component/table/BoolCell.tsx";
import {
	createDefaultTableOptions,
	filterValue,
	filterValueNumber,
	makeSchemaState,
	type OnChangeFn,
	setColumnFilter,
} from "../component/table/options.ts";
import { SortableTable } from "../component/table/SortableTable.tsx";
import type { VoteResult } from "../schema/votes.ts";
import { stringToColour } from "../util/colours.ts";
import { renderDateTime } from "../util/time.ts";

const validateSearch = makeSchemaState("vote_id");

export const Route = createFileRoute("/_mod/admin/votes")({
	component: AdminVotes,
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
		meta: [{ name: "description", content: "Votes" }, match.context.title("Votes")],
	}),
});

const columnHelper = createMRTColumnHelper<VoteResult>();
const defaultOptions = createDefaultTableOptions<VoteResult>();

function AdminVotes() {
	const servers = Route.useLoaderData();
	const search = Route.useSearch();
	const navigate = useNavigate();
	const theme = useTheme();

	const { data, isLoading, isError, isRefetching } = useQuery({
		queryKey: ["votes", { search }],
		queryFn: async ({ signal }) => {
			const server_id = filterValueNumber("server_id", search.columnFilters);
			const source_id = filterValue("source_id", search.columnFilters);
			const target_id = filterValue("target_id", search.columnFilters);
			const sort = search.sorting?.find((sort) => sort);

			return apiVotesQuery(
				{
					limit: search.pagination?.pageSize,
					offset: search.pagination ? search.pagination.pageIndex * search.pagination.pageSize : undefined,
					order_by: sort ? sort.id : "created_on",
					desc: sort ? sort.desc : true,
					source_id,
					target_id,
					server_id,
					success: -1,
				},
				signal,
			);
		},
	});

	const columns = useMemo(
		() => [
			columnHelper.accessor("server_id", {
				header: "Server",
				grow: false,
				filterVariant: "multi-select",
				filterSelectOptions: servers.map((server) => ({
					label: server.server_name,
					value: server.server_id,
				})),
				// filterFn: (row, _, filterValue) => {
				// 	return filterValue.length === 0 || filterValue.includes(row.original.server_id);
				// },
				Cell: ({ row, cell }) => (
					<Tooltip title={row.original.server_name}>
						<TextLink
							to={"/admin/votes"}
							search={setColumnFilter(search, "server_id", [cell.getValue()])}
							sx={{ color: stringToColour(row.original.server_name ?? "") }}
						>
							{row.original.server_name}
						</TextLink>
					</Tooltip>
				),
			}),
			columnHelper.accessor("source_id", {
				header: "Initiator",
				grow: true,
				enableSorting: false,
				Cell: ({ row }) => (
					<PersonCell
						steam_id={row.original.source_id}
						personaname={row.original.source_name}
						avatar_hash={row.original.source_avatar_hash}
					>
						<RouterLink
							style={{ color: theme.palette.primary.light }}
							to={Route.fullPath}
							search={setColumnFilter(search, "source_id", row.original.source_id)}
						>
							{row.original.source_name ?? row.original.source_id}
						</RouterLink>
					</PersonCell>
				),
			}),
			columnHelper.accessor("target_id", {
				header: "Subject",
				grow: true,
				enableSorting: false,
				Cell: ({ row }) => {
					return (
						<PersonCell
							steam_id={row.original.target_id}
							personaname={row.original.target_name}
							avatar_hash={row.original.target_avatar_hash}
						>
							<RouterLink
								style={{ color: theme.palette.primary.light }}
								to={Route.fullPath}
								search={setColumnFilter(search, "target_id", row.original.target_id)}
							>
								{row.original.target_name ?? row.original.target_id}
							</RouterLink>
						</PersonCell>
					);
				},
			}),
			columnHelper.accessor("success", {
				header: "Passed",
				grow: false,
				enableSorting: false,
				enableColumnFilter: false,
				filterVariant: "checkbox",
				Cell: ({ cell }) => {
					return <BoolCell enabled={cell.getValue()} />;
				},
			}),

			columnHelper.accessor("created_on", {
				header: "Created",
				grow: false,
				enableColumnFilter: false,
				Cell: ({ cell }) => renderDateTime(cell.getValue()),
			}),
		],
		[servers, search, theme],
	);

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

	const table = useMaterialReactTable({
		...defaultOptions,
		columns,
		data: data?.data ?? [],
		rowCount: data?.count ?? 0,
		enableFilters: true,
		state: {
			columnFilters: search.columnFilters,
			isLoading: isLoading || isRefetching,
			pagination: search.pagination,
			showAlertBanner: isError,
			showProgressBars: isRefetching,
			sorting: search.sorting,
		},
		manualFiltering: true,
		manualPagination: true,
		manualSorting: true,
		onColumnFiltersChange: setColumnFilters,
		onPaginationChange: setPagination,
		onSortingChange: setSorting,
		initialState: {
			...defaultOptions.initialState,
			columnVisibility: {
				source_id: true,
				target_id: true,
				passed: true,
				server_name: true,
			},
		},
	});

	return (
		<Grid container spacing={2}>
			<Grid size={{ xs: 12 }}>
				<SortableTable table={table} title={"Vote History"} />
			</Grid>
		</Grid>
	);
}
