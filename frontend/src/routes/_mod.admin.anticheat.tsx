/** biome-ignore-all lint/correctness/noChildrenProp: form needs it */

import { useTheme } from "@mui/material";
import Grid from "@mui/material/Grid";
import Tooltip from "@mui/material/Tooltip";
import Typography from "@mui/material/Typography";
import { createFileRoute, useNavigate } from "@tanstack/react-router";
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
import {
	createDefaultTableOptions,
	makeSchemaState,
	type OnChangeFn,
	setColumnFilter,
} from "../component/table/options.ts";
import { SortableTable } from "../component/table/SortableTable.tsx";
import { TableCellString } from "../component/table/TableCellString.tsx";
import { stringToColour } from "../util/colours.ts";
import { renderDate } from "../util/time.ts";
import { useQuery } from "@connectrpc/connect-query";
import { query } from "../rpc/anticheat/v1/anticheat-AnticheatService_connectquery.ts";
import type { Entry } from "../rpc/anticheat/v1/anticheat_pb.ts";

const validateSearch = makeSchemaState("anticheat_id");

export const Route = createFileRoute("/_mod/admin/anticheat")({
	component: AdminAnticheat,
	validateSearch,
	loader: async ({ context }) => {
		const unsorted = await context.queryClient.ensureQueryData({
			queryKey: ["serversSimple"],
			queryFn: async ({ signal }) => {
				return await apiGetServers(signal);
			},
		});
		return unsorted.sort((a, b) => {
			if (a.server_name > b.server_name) {
				return 1;
			}
			if (a.server_name < b.server_name) {
				return -1;
			}
			return 0;
		});
	},
	head: ({ match }) => ({
		meta: [{ name: "description", content: "Anti-Cheat Logs" }, match.context.title("Anti-Cheat Logs")],
	}),
});

const columnHelper = createMRTColumnHelper<Entry>();
const defaultOptions = createDefaultTableOptions<Entry>();

function AdminAnticheat() {
	const search = Route.useSearch();
	const navigate = useNavigate();
	const servers = Route.useLoaderData();
	const theme = useTheme();

	const { data, isLoading, isError } = useQuery(query);

	const columns = useMemo(
		() => [
			columnHelper.accessor("anticheatId", {
				header: "ID",
				enableSorting: false,
				enableColumnFilter: false,
				grow: false,
				Cell: ({ cell }) => <Typography>{cell.getValue()}</Typography>,
			}),
			columnHelper.accessor("serverId", {
				enableSorting: false,
				grow: false,
				enableColumnFilter: true,
				filterVariant: "multi-select",
				filterSelectOptions: servers.map((server) => ({
					label: server.server_name,
					value: server.server_id,
				})),
				filterFn: (row, _, filterValue) => {
					return filterValue.length === 0 || filterValue.includes(row.original.serverId);
				},
				header: "Server",
				Cell: ({ row, cell }) => (
					<Tooltip title={row.original.serverName}>
						<TextLink
							to={"/admin/anticheat"}
							search={setColumnFilter(search, "server_id", [cell.getValue()])}
							sx={{ color: stringToColour(row.original.serverName ?? "") }}
						>
							{row.original.serverName}
						</TextLink>
					</Tooltip>
				),
			}),
			columnHelper.accessor("steamId", {
				header: "Name",
				enableHiding: false,
				grow: true,
				Cell: ({ row }) => (
					<PersonCell
						steam_id={row.original.steamId}
						personaname={row.original.personaName}
						avatar_hash={row.original.avatarHash}
					>
						<RouterLink
							style={{
								color:
									theme.palette.mode === "dark"
										? theme.palette.primary.light
										: theme.palette.primary.dark,
							}}
							to={Route.fullPath}
							search={setColumnFilter(search, "steam_id", row.original.steamId)}
						>
							{row.original.personaName ?? row.original.serverId}
						</RouterLink>
					</PersonCell>
				),
			}),
			columnHelper.accessor("personaName", {
				enableHiding: true,
				grow: false,
				header: "Personaname",
			}),
			columnHelper.accessor("createdOn", {
				header: "Created",
				grow: false,
				Cell: ({ cell }) => renderDate(cell.getValue()),
			}),
			columnHelper.accessor("demoId", {
				header: "Demo",
				grow: false,
			}),
			columnHelper.accessor("detection", {
				header: "Detection",
				filterVariant: "multi-select",
				grow: false,
				Cell: ({ cell }) => (
					<TextLink to={"/admin/anticheat"} search={setColumnFilter(search, "detection", [cell.getValue()])}>
						{cell.getValue()}
					</TextLink>
				),
			}),
			columnHelper.accessor("triggered", {
				header: "Count",
				filterVariant: "range-slider",
				grow: false,
			}),
			columnHelper.accessor("summary", {
				header: "Summary",
				grow: true,
				Cell: ({ renderedCellValue }) => <TableCellString>{renderedCellValue}</TableCellString>,
			}),
		],
		[search, servers, theme],
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
		data: data?.entries ?? [],
		enableFilters: true,
		onColumnFiltersChange: setColumnFilters,
		onPaginationChange: setPagination,
		onSortingChange: setSorting,
		state: {
			isLoading,
			showAlertBanner: isError,
			pagination: search.pagination,
			sorting: search.sorting,
			columnFilters: search.columnFilters,
		},
		initialState: {
			...defaultOptions.initialState,
			columnVisibility: {
				anticheat_id: false,
				server_id: true,
				name: true,
				personaname: false,
				target_id: false,
				steam_id: true,
				demo_id: false,
				reason: true,
				reason_text: true,
				created_on: false,
			},
		},
	});

	return (
		<Grid container spacing={2}>
			<Grid size={{ xs: 12 }}>
				<SortableTable table={table} title={"Anti-Cheat Log Entries"} />
			</Grid>
		</Grid>
	);
}
