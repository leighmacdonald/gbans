import NiceModal from "@ebay/nice-modal-react";
import AddIcon from "@mui/icons-material/Add";
import EditIcon from "@mui/icons-material/Edit";
import StorageIcon from "@mui/icons-material/Storage";
import Button from "@mui/material/Button";
import ButtonGroup from "@mui/material/ButtonGroup";
import Grid from "@mui/material/Grid";
import IconButton from "@mui/material/IconButton";
import Stack from "@mui/material/Stack";
import Tooltip from "@mui/material/Tooltip";
import { useQueryClient } from "@tanstack/react-query";
import { createFileRoute } from "@tanstack/react-router";
import { createMRTColumnHelper, MaterialReactTable, useMaterialReactTable } from "material-react-table";
import { useMemo } from "react";
import { z } from "zod/v4";
import { apiGetServersAdmin } from "../api";
import { ContainerWithHeaderAndButtons } from "../component/ContainerWithHeaderAndButtons.tsx";
import { ServerEditorModal } from "../component/modal/ServerEditorModal.tsx";
import { BoolCell } from "../component/table/BoolCell.tsx";
import { createDefaultTableOptions } from "../component/table/options.ts";
import { TableCellString } from "../component/table/TableCellString.tsx";
import { TableCellStringHidden } from "../component/table/TableCellStringHidden.tsx";
import { useUserFlashCtx } from "../hooks/useUserFlashCtx.ts";
import type { Server } from "../schema/server.ts";
import { RowsPerPage } from "../util/table.ts";
import { renderDateTime } from "../util/time.ts";

const serversSearchSchema = z.object({
	page_index: z.number().optional().catch(0),
	page_size: z.number().optional().catch(RowsPerPage.TwentyFive),
	sort_order: z.enum(["desc", "asc"]).optional().catch("desc"),
	sort_column: z
		.enum(["server_id", "short_name", "name", "address", "port", "region", "cc", "enable_stats", "is_enabled"])
		.optional(),
});

export const Route = createFileRoute("/_admin/admin/servers")({
	validateSearch: (search) => serversSearchSchema.parse(search),
	loader: async ({ context }) => {
		const servers = await context.queryClient.fetchQuery({
			queryKey: ["serversAdmin"],
			queryFn: async () => {
				return (await apiGetServersAdmin()) ?? [];
			},
		});

		return { servers };
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
	const { servers } = Route.useLoaderData();
	const queryClient = useQueryClient();

	const onCreate = async () => {
		try {
			const newServer = (await NiceModal.show(ServerEditorModal, {})) as Server;
			queryClient.setQueryData(["serversAdmin"], [...(servers ?? []), newServer]);
			sendFlash("success", "Server created successfully");
		} catch (e) {
			sendFlash("error", `Failed to create new server: ${e}`);
		}
	};

	const onEdit = async (server: Server) => {
		try {
			const editedServer = (await NiceModal.show(ServerEditorModal, {
				server,
			})) as Server;
			queryClient.setQueryData(
				["serversAdmin"],
				servers.map((s) => {
					return s.server_id === editedServer.server_id ? editedServer : s;
				}),
			);
			sendFlash("success", "Server edited successfully");
		} catch (e) {
			sendFlash("error", `Failed to edit server: ${e}`);
		}
	};
	return (
		<Grid container spacing={2}>
			<Grid size={{ xs: 12 }}>
				<Stack spacing={2}>
					<ContainerWithHeaderAndButtons
						title={"Servers"}
						iconLeft={<StorageIcon />}
						buttons={[
							<ButtonGroup key={`server-header-buttons`}>
								<Button
									variant={"contained"}
									color={"success"}
									startIcon={<AddIcon />}
									sx={{ marginRight: 2 }}
									onClick={onCreate}
								>
									Create Server
								</Button>
							</ButtonGroup>,
						]}
					>
						<AdminServersTable servers={servers} isLoading={false} onEdit={onEdit} />
					</ContainerWithHeaderAndButtons>
				</Stack>
			</Grid>
		</Grid>
	);
}

const columnHelper = createMRTColumnHelper<Server>();
const defaultOptions = createDefaultTableOptions<Server>();

const AdminServersTable = ({
	servers,
	onEdit,
}: {
	servers: Server[];
	isLoading: boolean;
	onEdit: (server: Server) => Promise<void>;
}) => {
	const columns = useMemo(() => {
		return [
			columnHelper.accessor("server_id", {
				header: "ID",
				grow: false,
				size: 40,
				Cell: ({ cell }) => <TableCellString>{String(cell.getValue())}</TableCellString>,
			}),
			columnHelper.accessor("short_name", {
				size: 30,
				meta: {
					tooltip: "Short unique server identifier",
				},
				header: "Name",
				Cell: ({ cell }) => <TableCellString>{cell.getValue() as string}</TableCellString>,
			}),

			columnHelper.accessor("name", {
				size: 300,
				header: "Name Long",
				grow: true,
				meta: {
					tooltip: "Full name of the server, AKA srcds hostname",
				},
				Cell: ({ cell }) => <TableCellString>{cell.getValue() as string}</TableCellString>,
			}),

			columnHelper.accessor("address", {
				header: "Address",
				meta: {
					tooltip: "IP or DNS/Hostname of the server",
				},
				Cell: ({ cell }) => <TableCellString>{cell.getValue() as string}</TableCellString>,
			}),

			columnHelper.accessor("port", {
				header: "Port",
				size: 50,
				Cell: ({ cell }) => <TableCellString>{String(cell.getValue())}</TableCellString>,
			}),

			columnHelper.accessor("rcon", {
				header: "RCON",
				meta: {
					tooltip: "Standard RCON password",
				},
				Cell: ({ cell }) => <TableCellStringHidden>{cell.getValue() as string}</TableCellStringHidden>,
			}),

			columnHelper.accessor("password", {
				meta: {
					tooltip: "A password that the server uses to authenticate with the central gbans server",
				},
				header: "Auth Key",
				Cell: ({ cell }) => <TableCellStringHidden>{cell.getValue() as string}</TableCellStringHidden>,
			}),

			columnHelper.accessor("region", {
				header: "Region",
				size: 75,
				Cell: ({ cell }) => <TableCellString>{cell.getValue() as string}</TableCellString>,
			}),

			columnHelper.accessor("token_created_on", {
				meta: {
					tooltip: "Last time the server authenticated itself",
				},
				header: "Last Auth",
				Cell: ({ cell }) => <TableCellString>{renderDateTime(cell.getValue() as Date)}</TableCellString>,
			}),
			columnHelper.accessor("enable_stats", {
				size: 30,
				meta: {
					tooltip: "Stat Tracking Enabled",
				},
				header: "St",
				Cell: ({ cell }) => <BoolCell enabled={cell.getValue() as boolean} />,
			}),
			columnHelper.accessor("is_enabled", {
				size: 30,
				filterVariant: "checkbox",
				meta: {
					tooltip: "Enabled",
				},
				header: "En.",
				grow: false,
				Cell: ({ cell }) => <BoolCell enabled={cell.getValue() as boolean} />,
			}),
		];
	}, []);

	const table = useMaterialReactTable({
		...defaultOptions,
		columns,
		data: servers,
		enableFilters: true,
		initialState: {
			...defaultOptions.initialState,
			sorting: [{ id: "name", desc: false }],
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

	return <MaterialReactTable table={table} />;
};
