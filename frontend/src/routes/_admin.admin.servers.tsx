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
import {
	createColumnHelper,
	getCoreRowModel,
	getPaginationRowModel,
	type OnChangeFn,
	type PaginationState,
	useReactTable,
} from "@tanstack/react-table";
import { useMemo, useState } from "react";
import { z } from "zod/v4";
import { apiGetServersAdmin } from "../api";
import { ContainerWithHeaderAndButtons } from "../component/ContainerWithHeaderAndButtons.tsx";
import { PaginatorLocal } from "../component/forum/PaginatorLocal.tsx";
import { ServerEditorModal } from "../component/modal/ServerEditorModal.tsx";
import { BoolCell } from "../component/table/BoolCell.tsx";
import { DataTable } from "../component/table/DataTable.tsx";
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
				return await apiGetServersAdmin();
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

	const [pagination, setPagination] = useState({
		pageIndex: 0, //initial page index
		pageSize: RowsPerPage.TwentyFive, //default page size
	});

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
				(servers ?? []).map((s) => {
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
						<AdminServersTable
							servers={servers ?? []}
							isLoading={false}
							setPagination={setPagination}
							pagination={pagination}
							onEdit={onEdit}
						/>
						<PaginatorLocal
							onRowsChange={(rows) => {
								setPagination((prev) => {
									return { ...prev, pageSize: rows };
								});
							}}
							onPageChange={(page) => {
								setPagination((prev) => {
									return { ...prev, pageIndex: page };
								});
							}}
							count={servers?.length ?? 0}
							rows={pagination.pageSize}
							page={pagination.pageIndex}
						/>
					</ContainerWithHeaderAndButtons>
				</Stack>
			</Grid>
		</Grid>
	);
}

const columnHelper = createColumnHelper<Server>();

const AdminServersTable = ({
	servers,
	isLoading,
	setPagination,
	pagination,
	onEdit,
}: {
	servers: Server[];
	isLoading: boolean;
	onEdit: (server: Server) => Promise<void>;
	pagination: PaginationState;
	setPagination: OnChangeFn<PaginationState>;
}) => {
	const columns = useMemo(() => {
		return [
			columnHelper.accessor("server_id", {
				header: "ID",
				size: 40,
				cell: (info) => <TableCellString>{info.getValue()}</TableCellString>,
			}),
			columnHelper.accessor("short_name", {
				size: 60,
				meta: {
					tooltip: "Short unique server identifier",
				},
				header: "Name",
				cell: (info) => <TableCellString>{info.getValue()}</TableCellString>,
			}),

			columnHelper.accessor("name", {
				size: 300,
				header: "Name Long",
				meta: {
					tooltip: "Full name of the server, AKA srcds hostname",
				},
				cell: (info) => <TableCellString>{info.getValue()}</TableCellString>,
			}),

			columnHelper.accessor("address", {
				header: "Address",
				meta: {
					tooltip: "IP or DNS/Hostname of the server",
				},
				cell: (info) => <TableCellString>{info.getValue()}</TableCellString>,
			}),

			columnHelper.accessor("port", {
				header: "Port",
				size: 50,
				cell: (info) => <TableCellString>{info.getValue()}</TableCellString>,
			}),

			columnHelper.accessor("rcon", {
				header: "RCON",
				meta: {
					tooltip: "Standard RCON password",
				},
				cell: (info) => <TableCellStringHidden>{info.getValue()}</TableCellStringHidden>,
			}),

			columnHelper.accessor("password", {
				meta: {
					tooltip: "A password that the server uses to authenticate with the central gbans server",
				},
				header: () => "Auth Key",
				cell: (info) => <TableCellStringHidden>{info.getValue()}</TableCellStringHidden>,
			}),

			columnHelper.accessor("region", {
				header: "Region",
				size: 75,
				cell: (info) => <TableCellString>{info.getValue()}</TableCellString>,
			}),

			columnHelper.accessor("token_created_on", {
				meta: {
					tooltip: "Last time the server authenticated itself",
				},
				header: "Last Auth",
				cell: (info) => <TableCellString>{renderDateTime(info.getValue() as Date)}</TableCellString>,
			}),
			columnHelper.accessor("enable_stats", {
				size: 30,
				meta: {
					tooltip: "Stat Tracking Enabled",
				},
				header: "St",
				cell: (info) => <BoolCell enabled={info.getValue() as boolean} />,
			}),
			columnHelper.accessor("is_enabled", {
				size: 30,
				meta: {
					tooltip: "Enabled",
				},
				header: "En.",
				cell: (info) => <BoolCell enabled={info.getValue() as boolean} />,
			}),

			columnHelper.display({
				id: "actions",
				size: 30,
				meta: {
					tooltip: "Actions",
				},
				cell: (info) => {
					return (
						<ButtonGroup fullWidth variant={"text"}>
							<IconButton
								color={"warning"}
								onClick={async () => {
									await onEdit(info.row.original);
								}}
							>
								<Tooltip title={"Edit Server"}>
									<EditIcon />
								</Tooltip>
							</IconButton>
						</ButtonGroup>
					);
				},
			}),
		];
	}, [onEdit]);

	const table = useReactTable({
		data: servers,
		columns: columns,
		getCoreRowModel: getCoreRowModel(),
		getPaginationRowModel: getPaginationRowModel(),
		onPaginationChange: setPagination, //update the pagination state when internal APIs mutate the pagination state
		state: {
			pagination,
		},
	});

	return <DataTable table={table} isLoading={isLoading} />;
};
