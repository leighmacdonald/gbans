/** biome-ignore-all lint/correctness/noChildrenProp: form needs it */
import { useQuery } from "@connectrpc/connect-query";
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
import { Detection, type Entry } from "../rpc/anticheat/v1/anticheat_pb.ts";
import { query } from "../rpc/anticheat/v1/anticheat-AnticheatService_connectquery.ts";
import { servers } from "../rpc/servers/v1/servers-ServersService_connectquery.ts";
import { stringToColour } from "../util/colours.ts";
import { enumValues } from "../util/lists.ts";
import { detectionString } from "../util/strings.ts";
import { renderTimestamp } from "../util/time.ts";

const validateSearch = makeSchemaState("anticheatId");

export const Route = createFileRoute("/_mod/admin/anticheat")({
	component: AdminAnticheat,
	validateSearch,
	head: ({ match }) => ({
		meta: [{ name: "description", content: "Anti-Cheat Logs" }, match.context.title("Anti-Cheat Logs")],
	}),
});

const columnHelper = createMRTColumnHelper<Entry>();
const defaultOptions = createDefaultTableOptions<Entry>();

function AdminAnticheat() {
	const search = Route.useSearch();
	const navigate = useNavigate();
	const theme = useTheme();

	const { data: serverList, isLoading: isLoadingServers } = useQuery(servers);

	const { data, isLoading, isError } = useQuery(
		query,
		{
			steamId: 76561198084134025n,
		},
		{ enabled: !isLoadingServers },
	);

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
				filterSelectOptions: serverList?.servers.map((server) => ({
					label: server.serverName,
					value: server.serverId,
				})),
				filterFn: (row, _, filterValue) => {
					return filterValue.length === 0 || filterValue.includes(row.original.serverId);
				},
				header: "Server",
				Cell: ({ row, cell }) => (
					<Tooltip title={row.original.serverName}>
						<TextLink
							to={"/admin/anticheat"}
							search={setColumnFilter(search, "serverId", [cell.getValue()])}
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
						steamId={row.original.steamId}
						personaName={row.original.personaName}
						avatarHash={row.original.avatarHash}
					>
						<RouterLink
							style={{
								color:
									theme.palette.mode === "dark"
										? theme.palette.primary.light
										: theme.palette.primary.dark,
							}}
							to={Route.fullPath}
							search={setColumnFilter(search, "steamId", row.original.steamId)}
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
				Cell: ({ cell }) => renderTimestamp(cell.getValue()),
			}),
			columnHelper.accessor("demoId", {
				header: "Demo",
				grow: false,
			}),
			columnHelper.accessor("detection", {
				header: "Detection",
				enableColumnFilter: true,
				filterVariant: "multi-select",
				filterSelectOptions: enumValues(Detection).map((server) => ({
					label: detectionString(server),
					value: server,
				})),
				grow: false,
				Cell: ({ cell }) => (
					<TextLink to={"/admin/anticheat"} search={setColumnFilter(search, "detection", [cell.getValue()])}>
						{detectionString(cell.getValue())}
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
				anticheatId: false,
				serverId: true,
				name: true,
				personaname: false,
				targetId: false,
				steamId: true,
				demoId: false,
				reason: true,
				reasonText: true,
				createdOn: false,
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
