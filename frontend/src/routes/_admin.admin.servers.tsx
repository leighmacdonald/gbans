import { useQuery } from "@connectrpc/connect-query";
import NiceModal from "@ebay/nice-modal-react";
import AddIcon from "@mui/icons-material/Add";
import EditIcon from "@mui/icons-material/Edit";
import ListIcon from "@mui/icons-material/List";
import Grid from "@mui/material/Grid";
import IconButton from "@mui/material/IconButton";
import Tooltip from "@mui/material/Tooltip";
import Typography from "@mui/material/Typography";
import { useQueryClient } from "@tanstack/react-query";
import { createFileRoute, stripSearchParams, useNavigate } from "@tanstack/react-router";
import {
	createMRTColumnHelper,
	type MRT_ColumnFiltersState,
	type MRT_PaginationState,
	type MRT_SortingState,
	useMaterialReactTable,
} from "material-react-table";
import { useCallback, useMemo } from "react";
import { IconButtonLink } from "../component/IconButtonLink.tsx";
import { ServerEditorModal } from "../component/modal/ServerEditorModal.tsx";
import { RowActionContainer } from "../component/RowActionContainer.tsx";
import { BoolCell } from "../component/table/BoolCell.tsx";
import {
	createDefaultTableOptions,
	makeRowActionsDefOptions,
	makeSchemaDefaults,
	makeSchemaState,
	type OnChangeFn,
} from "../component/table/options.ts";
import { SortableTable } from "../component/table/SortableTable.tsx";
import { TableCellStringHidden } from "../component/table/TableCellStringHidden.tsx";
import { useUserFlashCtx } from "../hooks/useUserFlashCtx.ts";
import type { Server } from "../rpc/servers/v1/servers_pb.ts";
import { serversAdmin } from "../rpc/servers/v1/servers-ServersService_connectquery.ts";
import { stringToColour } from "../util/colours.ts";
import { renderTimestamp } from "../util/time.ts";

const columnHelper = createMRTColumnHelper<Server>();
const defaultOptions = createDefaultTableOptions<Server>();
const defaultValues = {
	...makeSchemaDefaults({ defaultColumn: "name" }),
	columnFilters: [{ id: "is_enabled", value: "true" }],
	sorting: [{ id: "name", desc: false }],
	pagination: {
		pageIndex: 0,
		pageSize: 25,
	},
};
const validateSearch = makeSchemaState("name", false);

export const Route = createFileRoute("/_admin/admin/servers")({
	validateSearch,
	search: {
		middlewares: [stripSearchParams(defaultValues)],
	},
	head: ({ match }) => {
		return {
			meta: [{ name: "description", content: "Server Editor" }, match.context.title("Edit Servers")],
		};
	},
	component: AdminServers,
});

function AdminServers() {
	const { sendFlash } = useUserFlashCtx();
	const queryClient = useQueryClient();
	const search = Route.useSearch();
	const navigate = useNavigate();

	const { data, isLoading, isError } = useQuery(serversAdmin);

	const onCreate = useCallback(async () => {
		try {
			const newServer = (await NiceModal.show(ServerEditorModal, {})) as Server;
			queryClient.setQueryData(["serversAdmin"], [...(data?.servers ?? []), newServer]);
			sendFlash("success", "Server created successfully");
		} catch (e) {
			sendFlash("error", `Failed to create new server: ${e}`);
		}
	}, [data, sendFlash, queryClient]);

	const onEdit = useCallback(
		async (server: Server) => {
			try {
				const editedServer = (await NiceModal.show(ServerEditorModal, {
					server,
				})) as Server;
				queryClient.setQueryData(
					["serversAdmin"],
					(data?.servers ?? []).map((s) => {
						return s.serverId === editedServer.serverId ? editedServer : s;
					}),
				);
				sendFlash("success", "Server edited successfully");
			} catch (e) {
				sendFlash("error", `Failed to edit server: ${e}`);
			}
		},
		[data, sendFlash, queryClient],
	);

	const columns = useMemo(() => {
		return [
			columnHelper.accessor("serverId", {
				header: "ID",
				grow: false,
			}),

			columnHelper.accessor("shortName", {
				grow: false,
				meta: {
					tooltip: "Short unique server identifier",
				},
				header: "Name",
				Cell: ({ cell, row }) => (
					<Typography sx={{ color: stringToColour(row.original.shortName) }}>{cell.getValue()}</Typography>
				),
			}),

			columnHelper.accessor("name", {
				header: "Name Long",
				grow: true,
				meta: {
					tooltip: "Full name of the server, AKA srcds hostname",
				},
				Cell: ({ cell, row }) => (
					<Typography sx={{ color: stringToColour(row.original.shortName) }}>{cell.getValue()}</Typography>
				),
			}),

			columnHelper.accessor("address", {
				header: "Address",
				grow: false,
				meta: {
					tooltip: "IP or DNS/Hostname of the server",
				},
			}),

			columnHelper.accessor("addressInternal", {
				header: "Internal Addr",
				grow: false,
				meta: {
					tooltip: "Internal IP or DNS/Hostname",
				},
			}),

			columnHelper.accessor("port", {
				header: "Port",
				grow: false,
			}),

			columnHelper.accessor("rcon", {
				header: "RCON",
				meta: {
					tooltip: "Standard RCON password",
				},
				grow: false,
				Cell: ({ cell }) => <TableCellStringHidden>{cell.getValue() as string}</TableCellStringHidden>,
			}),

			columnHelper.accessor("password", {
				meta: {
					tooltip: "A password that the server uses to authenticate with the central gbans server",
				},
				header: "Auth Key",
				grow: false,
				Cell: ({ cell }) => <TableCellStringHidden>{cell.getValue() as string}</TableCellStringHidden>,
			}),

			columnHelper.accessor("region", {
				header: "Region",
				grow: false,
			}),

			columnHelper.accessor("tokenCreatedOn", {
				meta: {
					tooltip: "Last time the server authenticated itself",
				},
				header: "Last Auth",
				grow: false,
				Cell: ({ cell }) => renderTimestamp(cell.getValue()),
			}),

			columnHelper.accessor("enableStats", {
				meta: {
					tooltip: "Stat Tracking Enabled",
				},
				filterVariant: "checkbox",
				header: "Stats",
				grow: false,
				Cell: ({ cell }) => <BoolCell enabled={cell.getValue() as boolean} />,
			}),

			columnHelper.accessor("isEnabled", {
				filterVariant: "checkbox",
				meta: {
					tooltip: "Enabled",
				},
				header: "Enabled",
				grow: false,
				Cell: ({ cell }) => <BoolCell enabled={cell.getValue() as boolean} />,
			}),
		];
	}, []);

	const setSorting: OnChangeFn<MRT_SortingState> = useCallback(
		async (updater) => {
			await navigate({
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
		async (updater) => {
			await navigate({
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
		async (updater) => {
			await navigate({
				to: Route.fullPath,
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
		[search, navigate],
	);
	const table = useMaterialReactTable({
		...defaultOptions,
		columns,
		data: data?.servers ?? [],
		enableFilters: true,
		onColumnFiltersChange: setColumnFilters,
		onPaginationChange: setPagination,
		onSortingChange: setSorting,
		state: {
			isLoading,
			showAlertBanner: isError,
			columnFilters: search.columnFilters,
			sorting: search.sorting,
			pagination: search.pagination,
		},
		initialState: {
			columnFilters: defaultValues.columnFilters,
			pagination: defaultValues.pagination,
			sorting: defaultValues.sorting,
			columnVisibility: {
				server_id: false,
				short_name: true,
				password: false,
				region: false,
				rcon: false,
				token_created_on: false,
				enable_stats: false,
				is_enabled: true,
			},
		},
		enableRowActions: true,
		displayColumnDefOptions: makeRowActionsDefOptions(2),
		renderRowActions: ({ row }) => (
			<RowActionContainer>
				<IconButton
					key="edit"
					color={"warning"}
					onClick={async () => {
						await onEdit(row.original);
					}}
				>
					<Tooltip title={"Edit Server"}>
						<EditIcon />
					</Tooltip>
				</IconButton>
				<IconButtonLink
					to={"/admin/serverlogs"}
					search={{ server_ids: [row.original.serverId] }}
					key="logs"
					color={"warning"}
				>
					<Tooltip title={"Server Logs"}>
						<ListIcon />
					</Tooltip>
				</IconButtonLink>
			</RowActionContainer>
		),
	});

	return (
		<Grid container spacing={2}>
			<Grid size={{ xs: 12 }}>
				<SortableTable
					table={table}
					title={"Servers"}
					buttons={[
						<IconButton key="create" onClick={onCreate} sx={{ color: "primary.contrastText" }}>
							<AddIcon />
						</IconButton>,
					]}
				/>
			</Grid>
		</Grid>
	);
}
