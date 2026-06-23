/** biome-ignore-all lint/correctness/noChildrenProp: form needs it */

import { create } from "@bufbuild/protobuf";
import { useQuery } from "@connectrpc/connect-query";
import Grid from "@mui/material/Grid";
import { useTheme } from "@mui/material/styles";
import Tooltip from "@mui/material/Tooltip";
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
import { TextLink } from "../component/TextLink.tsx";
import { BoolCell } from "../component/table/BoolCell.tsx";
import {
	createDefaultTableOptions,
	filterValueNumber,
	filterValueString,
	makeSchemaDefaults,
	makeSchemaState,
	type OnChangeFn,
	setColumnFilter,
} from "../component/table/options.ts";
import { SortableTable } from "../component/table/SortableTable.tsx";
import { renderTableError } from "../error.tsx";
import { servers } from "../rpc/servers/v1/servers-ServersService_connectquery.ts";
import { QueryRequestSchema, type VoteResult } from "../rpc/votes/v1/votes_pb.ts";
import { query } from "../rpc/votes/v1/votes-VotesService_connectquery.ts";
import { stringToColour } from "../util/colours.ts";
import { renderTimestamp } from "../util/time.ts";
import { emptyOrNullString } from "../util/types.ts";

const columnHelper = createMRTColumnHelper<VoteResult>();
const defaultOptions = createDefaultTableOptions<VoteResult>();
const validateSearch = makeSchemaState("voteId");
const defaultValues = makeSchemaDefaults({ defaultColumn: "voteId" });

export const Route = createFileRoute("/_mod/admin/votes")({
	component: AdminVotes,
	validateSearch,
	search: {
		middlewares: [stripSearchParams(defaultValues)],
	},
	head: ({ match }) => ({
		meta: [{ name: "description", content: "Votes" }, match.context.title("Votes")],
	}),
});

function AdminVotes() {
	const search = Route.useSearch();
	const navigate = useNavigate();
	const theme = useTheme();
	const { data: serverList, isLoading: isLoadingServers } = useQuery(servers);

	const opts = useMemo(() => {
		const serverId = filterValueNumber("serverId", search.columnFilters);
		const sourceId = filterValueString("sourceId", search.columnFilters);
		const targetId = filterValueString("targetId", search.columnFilters);
		const opts = create(QueryRequestSchema, {
			filter: {
				limit: String(search.pagination?.pageSize ?? 25),
				offset: String(search.pagination ? search.pagination.pageIndex * search.pagination.pageSize : 0),
				orderBy: search.sorting?.find((sort) => sort)?.id ?? "createdOn",
				desc: search.sorting?.find((sort) => sort)?.desc ?? true,
			},
		});
		if (serverId > 0) {
			opts.serverId = serverId;
		}

		if (!emptyOrNullString(sourceId)) {
			opts.sourceId = sourceId;
		}

		if (!emptyOrNullString(targetId)) {
			opts.targetId = targetId;
		}

		return opts;
	}, [search]);

	const { data, isLoading, isError, isRefetching, error } = useQuery(query, opts);

	const columns = useMemo(
		() => [
			columnHelper.accessor("serverId", {
				header: "Server",
				grow: false,
				filterVariant: "multi-select",
				filterSelectOptions: serverList?.servers.map((server) => ({
					label: server.serverName,
					value: server.serverId,
				})),
				// filterFn: (row, _, filterValue) => {
				// 	return filterValue.length === 0 || filterValue.includes(row.original.server_id);
				// },
				Cell: ({ row, cell }) => (
					<Tooltip title={row.original.serverName}>
						<TextLink
							to={"/admin/votes"}
							search={setColumnFilter(search, "serverId", [cell.getValue()])}
							sx={{ color: stringToColour(row.original.serverName ?? "") }}
						>
							{row.original.serverName}
						</TextLink>
					</Tooltip>
				),
			}),
			columnHelper.accessor("sourceId", {
				header: "Initiator",
				grow: true,
				enableSorting: false,
				Cell: ({ row }) => (
					<PersonCell
						steamId={row.original.sourceId.toString()}
						personaName={row.original.sourceName}
						avatarHash={row.original.sourceAvatarHash}
					>
						<RouterLink
							style={{
								color:
									theme.palette.mode === "dark"
										? theme.palette.primary.light
										: theme.palette.primary.dark,
							}}
							to={Route.fullPath}
							search={setColumnFilter(search, "sourceId", row.original.sourceId)}
						>
							{row.original.sourceName ?? row.original.sourceId}
						</RouterLink>
					</PersonCell>
				),
			}),
			columnHelper.accessor("targetId", {
				header: "Subject",
				grow: true,
				enableSorting: false,
				Cell: ({ row }) => {
					return (
						<PersonCell
							steamId={row.original.targetId.toString()}
							personaName={row.original.targetName}
							avatarHash={row.original.targetAvatarHash}
						>
							<RouterLink
								style={{
									color:
										theme.palette.mode === "dark"
											? theme.palette.primary.light
											: theme.palette.primary.dark,
								}}
								to={Route.fullPath}
								search={setColumnFilter(search, "targeId", row.original.targetId)}
							>
								{row.original.targetName ?? row.original.targetId}
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

			columnHelper.accessor("createdOn", {
				header: "Created",
				grow: false,
				enableColumnFilter: false,
				Cell: ({ cell }) => renderTimestamp(cell.getValue()),
			}),
		],
		[search, theme, serverList?.servers],
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
		data: data?.results ?? [],
		rowCount: Number(data?.count ?? 0),
		enableFilters: true,
		state: {
			columnFilters: search.columnFilters,
			isLoading: isLoadingServers || isLoading || isRefetching,
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
		muiToolbarAlertBannerProps: renderTableError(error),
		initialState: {
			...defaultOptions.initialState,
			columnVisibility: {
				sourceId: true,
				targetId: true,
				passed: true,
				serverName: true,
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
