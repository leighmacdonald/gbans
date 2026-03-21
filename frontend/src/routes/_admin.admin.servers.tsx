import NiceModal from "@ebay/nice-modal-react";
import AddIcon from "@mui/icons-material/Add";
import EditIcon from "@mui/icons-material/Edit";
import Grid from "@mui/material/Grid";
import IconButton from "@mui/material/IconButton";
import Tooltip from "@mui/material/Tooltip";
import Typography from "@mui/material/Typography";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { createFileRoute } from "@tanstack/react-router";
import { createMRTColumnHelper, useMaterialReactTable } from "material-react-table";
import { useCallback, useMemo } from "react";
import { apiGetServersAdmin } from "../api";
import { ServerEditorModal } from "../component/modal/ServerEditorModal.tsx";
import { BoolCell } from "../component/table/BoolCell.tsx";
import { createDefaultTableOptions } from "../component/table/options.ts";
import { SortableTable } from "../component/table/SortableTable.tsx";
import { TableCellStringHidden } from "../component/table/TableCellStringHidden.tsx";
import { useUserFlashCtx } from "../hooks/useUserFlashCtx.ts";
import type { Server } from "../schema/server.ts";
import { stringToColour } from "../util/colours.ts";
import { renderDateTime } from "../util/time.ts";

const columnHelper = createMRTColumnHelper<Server>();
const defaultOptions = createDefaultTableOptions<Server>();

export const Route = createFileRoute("/_admin/admin/servers")({
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

	const { data, isLoading, isError } = useQuery({
		queryKey: ["serversAdmin"],
		queryFn: async () => {
			return (await apiGetServersAdmin()) ?? [];
		},
	});

	const onCreate = useCallback(async () => {
		try {
			const newServer = (await NiceModal.show(ServerEditorModal, {})) as Server;
			queryClient.setQueryData(["serversAdmin"], [...(data ?? []), newServer]);
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
					(data ?? []).map((s) => {
						return s.server_id === editedServer.server_id ? editedServer : s;
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
			columnHelper.accessor("server_id", {
				header: "ID",
				grow: false,
			}),
			columnHelper.accessor("short_name", {
				grow: false,
				meta: {
					tooltip: "Short unique server identifier",
				},
				header: "Name",
				Cell: ({ cell, row }) => (
					<Typography sx={{ color: stringToColour(row.original.short_name) }}>{cell.getValue()}</Typography>
				),
			}),

			columnHelper.accessor("name", {
				header: "Name Long",
				grow: true,
				meta: {
					tooltip: "Full name of the server, AKA srcds hostname",
				},
				Cell: ({ cell, row }) => (
					<Typography sx={{ color: stringToColour(row.original.short_name) }}>{cell.getValue()}</Typography>
				),
			}),

			columnHelper.accessor("address", {
				header: "Address",
				grow: false,
				meta: {
					tooltip: "IP or DNS/Hostname of the server",
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

			columnHelper.accessor("token_created_on", {
				meta: {
					tooltip: "Last time the server authenticated itself",
				},
				header: "Last Auth",
				grow: false,
				Cell: ({ cell }) => renderDateTime(cell.getValue() as Date),
			}),
			columnHelper.accessor("enable_stats", {
				meta: {
					tooltip: "Stat Tracking Enabled",
				},
				header: "Stats",
				grow: false,
				Cell: ({ cell }) => <BoolCell enabled={cell.getValue() as boolean} />,
			}),
			columnHelper.accessor("is_enabled", {
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

	const table = useMaterialReactTable({
		...defaultOptions,
		columns,
		data: data ?? [],
		enableFilters: true,
		state: {
			isLoading,
			showAlertBanner: isError,
		},
		initialState: {
			...defaultOptions.initialState,
			sorting: [{ id: "name", desc: false }],
			pagination: {
				pageIndex: 0,
				pageSize: 100,
			},
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
		renderRowActionMenuItems: ({ row }) => [
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
			</IconButton>,
		],
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
