import NiceModal, { useModal } from "@ebay/nice-modal-react";
import AddIcon from "@mui/icons-material/Add";
import DeleteIcon from "@mui/icons-material/Delete";
import EditIcon from "@mui/icons-material/Edit";
import PersonOffIcon from "@mui/icons-material/PersonOff";
import PublicOffIcon from "@mui/icons-material/PublicOff";
import WifiOffIcon from "@mui/icons-material/WifiOff";
import Button from "@mui/material/Button";
import ButtonGroup from "@mui/material/ButtonGroup";
import Grid from "@mui/material/Grid";
import Stack from "@mui/material/Stack";
import TableCell from "@mui/material/TableCell";
import Typography from "@mui/material/Typography";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { createFileRoute } from "@tanstack/react-router";
import { createColumnHelper, getCoreRowModel, useReactTable } from "@tanstack/react-table";
import { useCallback, useMemo } from "react";
import { z } from "zod/v4";
import {
	apiDeleteCIDRBlockSource,
	apiDeleteCIDRBlockWhitelist,
	apiDeleteWhitelistSteam,
	apiGetCIDRBlockLists,
	apiGetCIDRBlockListsIPWhitelist,
	apiGetCIDRBlockListsSteamWhitelist,
} from "../api";
import { ContainerWithHeaderAndButtons } from "../component/ContainerWithHeaderAndButtons.tsx";
import { CIDRBlockEditorModal } from "../component/modal/CIDRBlockEditorModal.tsx";
import { ConfirmationModal } from "../component/modal/ConfirmationModal.tsx";
import { IPWhitelistEditorModal } from "../component/modal/IPWhitelistEditorModal.tsx";
import { SteamWhitelistEditorModal } from "../component/modal/SteamWhitelistEditorModal.tsx";
import { PersonCell } from "../component/PersonCell.tsx";
import { DataTable } from "../component/table/DataTable.tsx";
import { VCenterBox } from "../component/VCenterBox.tsx";
import { useAuth } from "../hooks/useAuth.ts";
import { useUserFlashCtx } from "../hooks/useUserFlashCtx.ts";
import type { CIDRBlockSource, WhitelistIP, WhitelistSteam } from "../schema/network.ts";
import { PermissionLevel } from "../schema/people.ts";
import { logErr } from "../util/errors.ts";
import { renderDate } from "../util/time.ts";

const ipHistorySearchSchema = z.object({
	sortColumn: z.enum(["person_connection_id", "steam_id", "created_on", "ip_addr", "server_id"]).optional(),
});

export const Route = createFileRoute("/_mod/admin/network/cidrblocks")({
	component: AdminNetworkCIDRBlocks,
	head: () => ({
		meta: [{ name: "description", content: "CIDR Network Blocks" }, { title: "CIDR Network Blocks" }],
	}),
	validateSearch: (search) => ipHistorySearchSchema.parse(search),
});

function AdminNetworkCIDRBlocks() {
	const queryClient = useQueryClient();
	const { sendFlash, sendError } = useUserFlashCtx();
	const confirmModal = useModal(ConfirmationModal);
	const { hasPermission } = useAuth();

	const { data: blockSources, isLoading: isLoadingBlockSources } = useQuery({
		queryKey: ["networkBlockListSources"],
		queryFn: async () => {
			return await apiGetCIDRBlockLists();
		},
	});

	const { data: ipWhitelist, isLoading: isLoadingIPWhitelist } = useQuery({
		queryKey: ["networkIPWhitelist"],
		queryFn: async () => {
			return await apiGetCIDRBlockListsIPWhitelist();
		},
	});

	const { data: steamWhitelist, isLoading: isLoadingSteamWhitelist } = useQuery({
		queryKey: ["networkSteamWhitelist"],
		queryFn: async () => {
			return await apiGetCIDRBlockListsSteamWhitelist();
		},
	});

	const onIPWhitelistDeleteEdit = useCallback(
		async (source?: WhitelistIP) => {
			try {
				const newSource = (await NiceModal.show(IPWhitelistEditorModal, {
					source,
				})) as WhitelistIP;

				queryClient.setQueryData(
					["networkBlockListSourcesAdd"],
					(ipWhitelist ?? []).map((src) => {
						return src.cidr_block_whitelist_id === newSource.cidr_block_whitelist_id ? newSource : src;
					}),
				);
				sendFlash("success", "IP whitelist added");
			} catch (e) {
				sendFlash("error", `Failed to delete ip whitelist: ${e}`);
			}
		},
		[ipWhitelist, queryClient, sendFlash],
	);

	const ipWhitelistMutation = useMutation({
		mutationKey: ["networkIPWhitelistDelete"],
		mutationFn: async (variables: { cidr_block_whitelist_id: number }) => {
			await apiDeleteCIDRBlockWhitelist(variables.cidr_block_whitelist_id);
		},
		onSuccess: () => {
			sendFlash("success", "IP whitelist deleted");
		},
		onError: sendError,
	});

	const onIPWhitelistDelete = useCallback(
		async (source: WhitelistIP) => {
			try {
				const confirmed = await confirmModal.show({
					title: "Delete CIDR Whitelist?",
					children: "This action is permanent",
				});
				if (confirmed) {
					ipWhitelistMutation.mutate({
						cidr_block_whitelist_id: source.cidr_block_whitelist_id,
					});
				}
				await confirmModal.hide();
			} catch (e) {
				logErr(e);
			}
		},
		[ipWhitelistMutation, confirmModal],
	);

	const sourceMutation = useMutation({
		mutationKey: ["networkBlockSourceDelete"],
		mutationFn: async (variables: { cidr_block_source_id: number }) => {
			await apiDeleteCIDRBlockSource(variables.cidr_block_source_id);
		},
		onSuccess: (_, variables) => {
			sendFlash("success", "Blocklist source deleted");
			queryClient.setQueryData(
				["networkBlockListSources"],
				blockSources?.filter((b) => b.cidr_block_source_id !== variables.cidr_block_source_id),
			);
		},
		onError: sendError,
	});

	const onDeleteSource = useCallback(
		async (cidr_block_source_id: number) => {
			try {
				const confirmed = await confirmModal.show({
					title: "Delete CIDR Block Source?",
					children: "This action is permanent",
				});
				if (confirmed) {
					sourceMutation.mutate({ cidr_block_source_id });
				}
				await confirmModal.hide();
			} catch (e) {
				logErr(e);
			}
		},
		[confirmModal, sourceMutation],
	);

	const onEditBlockSource = useCallback(
		async (source?: CIDRBlockSource) => {
			try {
				const updated = (await NiceModal.show(CIDRBlockEditorModal, {
					source,
				})) as CIDRBlockSource;

				queryClient.setQueryData(
					["networkBlockListSources"],
					(blockSources ?? []).map((bs) => {
						return bs.cidr_block_source_id === updated.cidr_block_source_id ? updated : bs;
					}),
				);
			} catch (e) {
				logErr(e);
			}
		},
		[blockSources, queryClient],
	);

	const steamWhitelistDelete = useMutation({
		mutationKey: ["networkSteamWhitelistDelete"],
		mutationFn: async (variables: { steam_id: string }) => {
			await apiDeleteWhitelistSteam(variables.steam_id);
		},
		onSuccess: () => {
			sendFlash("success", "Steam whitelist deleted");
		},
		onError: sendError,
	});

	const onSteamWhitelistEdit = useCallback(async () => {
		try {
			const newSource = (await NiceModal.show(SteamWhitelistEditorModal, {})) as WhitelistSteam;

			queryClient.setQueryData(
				["networkSteamWhitelist"],
				(steamWhitelist ?? []).map((src) => {
					return src.steam_id === newSource.steam_id ? newSource : src;
				}),
			);
			sendFlash("success", "Steam whitelist added");
		} catch (e) {
			sendFlash("error", `Failed to add steam whitelist: ${e}`);
		}
	}, [queryClient, sendFlash, steamWhitelist]);

	const onSteamWhitelistDelete = useCallback(
		async (wl: WhitelistSteam) => {
			try {
				const confirmed = await confirmModal.show({
					title: "Delete steam whitelist?",
					children: "This action is permanent",
				});
				if (confirmed) {
					steamWhitelistDelete.mutate({ steam_id: wl.steam_id });
				}
				await confirmModal.hide();
			} catch (e) {
				logErr(e);
			}
		},
		[confirmModal, steamWhitelistDelete],
	);

	return (
		<Stack spacing={2}>
			<ContainerWithHeaderAndButtons
				title="Admin Network CIDR"
				iconLeft={<WifiOffIcon />}
				buttons={[
					<ButtonGroup size={"small"} key={"heading-icons"}>
						<Button
							startIcon={<AddIcon />}
							variant={"contained"}
							color={"success"}
							onClick={async () => {
								await onEditBlockSource();
							}}
						>
							New Blocklist
						</Button>
					</ButtonGroup>,
				]}
			>
				<Grid container spacing={2}>
					<Grid size={{ xs: 12 }}>
						<Stack spacing={1}>
							{!isLoadingBlockSources &&
								(blockSources ?? []).map((s) => {
									return (
										<Stack
											spacing={1}
											direction={"row"}
											key={`cidr-source-${s.cidr_block_source_id}`}
										>
											<ButtonGroup
												size={"small"}
												disabled={!hasPermission(PermissionLevel.Admin)}
											>
												<Button
													startIcon={<EditIcon />}
													variant={"contained"}
													color={"warning"}
													disabled={!hasPermission(PermissionLevel.Admin)}
													onClick={async () => {
														await onEditBlockSource(s);
													}}
												>
													Edit
												</Button>
												<Button
													startIcon={<DeleteIcon />}
													variant={"contained"}
													color={"error"}
													onClick={async () => {
														await onDeleteSource(s.cidr_block_source_id);
													}}
												>
													Delete
												</Button>
											</ButtonGroup>

											<VCenterBox>
												<Typography variant={"body1"}>{s.name}</Typography>
											</VCenterBox>
											<VCenterBox>
												<Typography variant={"body2"}>
													{s.enabled ? "Enabled" : "Disabled"}
												</Typography>
											</VCenterBox>
											<VCenterBox>
												<Typography variant={"body2"}>{s.url}</Typography>
											</VCenterBox>
										</Stack>
									);
								})}
						</Stack>
					</Grid>
				</Grid>
			</ContainerWithHeaderAndButtons>

			<ContainerWithHeaderAndButtons
				title={"IP Whitelists"}
				iconLeft={<PublicOffIcon />}
				buttons={[
					<ButtonGroup size={"small"} key="ip-whitelist-buttons">
						<Button
							startIcon={<AddIcon />}
							variant={"contained"}
							color={"success"}
							onClick={async () => {
								await onIPWhitelistDeleteEdit();
							}}
						>
							New IP Whitelist
						</Button>
					</ButtonGroup>,
				]}
			>
				<Grid container spacing={1}>
					<Grid size={{ xs: 12 }}>
						<IPWhitelistTable
							whitelist={ipWhitelist ?? []}
							isLoading={isLoadingIPWhitelist}
							onEdit={onIPWhitelistDeleteEdit}
							onDelete={onIPWhitelistDelete}
						/>
					</Grid>
				</Grid>
			</ContainerWithHeaderAndButtons>

			<ContainerWithHeaderAndButtons
				title={"Steam Whitelists"}
				iconLeft={<PersonOffIcon />}
				buttons={[
					<ButtonGroup size={"small"} key={"steam-whitelist-buttons"}>
						<Button
							startIcon={<AddIcon />}
							variant={"contained"}
							color={"success"}
							onClick={async () => {
								await onSteamWhitelistEdit();
							}}
						>
							New Steam Whitelist
						</Button>
					</ButtonGroup>,
				]}
			>
				<Grid container spacing={1}>
					<Grid size={{ xs: 12 }}>
						<SteamWhitelistTable
							whitelist={steamWhitelist ?? []}
							isLoading={isLoadingSteamWhitelist}
							onDelete={onSteamWhitelistDelete}
						/>
					</Grid>
				</Grid>
			</ContainerWithHeaderAndButtons>
		</Stack>
	);
}

const IPWhitelistTable = ({
	whitelist,
	isLoading,
	onEdit,
	onDelete,
}: {
	whitelist: WhitelistIP[];
	isLoading: boolean;
	onEdit: (wl: WhitelistIP) => Promise<void>;
	onDelete: (wl: WhitelistIP) => Promise<void>;
}) => {
	const columnHelper = createColumnHelper<WhitelistIP>();

	const columns = useMemo(
		() => [
			columnHelper.accessor("cidr_block_whitelist_id", {
				header: "ID",
				size: 30,
				cell: (info) => <Typography>{info.getValue() as number}</Typography>,
			}),
			columnHelper.accessor("address", {
				header: "Address",
				size: 100,
				cell: (info) => (
					<TableCell>
						<Typography>{info.getValue()}</Typography>
					</TableCell>
				),
			}),
			columnHelper.accessor("created_on", {
				header: "IP Address",
				cell: (info) => (
					<TableCell>
						<Typography>{renderDate(info.getValue() as Date)}</Typography>
					</TableCell>
				),
			}),
			columnHelper.accessor("updated_on", {
				header: "Updated",
				size: 500,
				cell: (info) => (
					<TableCell>
						<Typography>{renderDate(info.getValue() as Date)}</Typography>
					</TableCell>
				),
			}),
			columnHelper.display({
				id: "actions",
				header: "Actions",
				size: 150,
				cell: (info) => (
					<ButtonGroup variant={"contained"}>
						<Button
							startIcon={<EditIcon />}
							color={"warning"}
							onClick={async () => {
								await onEdit(info.row.original);
							}}
						>
							Edit
						</Button>

						<Button
							startIcon={<DeleteIcon />}
							color={"error"}
							onClick={async () => {
								await onDelete(info.row.original);
							}}
						>
							Delete
						</Button>
					</ButtonGroup>
				),
			}),
		],
		[columnHelper, onDelete, onEdit],
	);

	const table = useReactTable({
		data: whitelist,
		columns: columns,
		getCoreRowModel: getCoreRowModel(),
		manualPagination: true,
		autoResetPageIndex: true,
	});

	return <DataTable table={table} isLoading={isLoading} />;
};

const SteamWhitelistTable = ({
	whitelist,
	isLoading,
	onDelete,
}: {
	whitelist: WhitelistSteam[];
	isLoading: boolean;
	onDelete: (wl: WhitelistSteam) => Promise<void>;
}) => {
	const columnHelper = createColumnHelper<WhitelistSteam>();

	const columns = useMemo(
		() => [
			columnHelper.accessor("steam_id", {
				header: "Steam ID",
				size: 600,
				cell: (info) => (
					<PersonCell
						steam_id={info.row.original.steam_id}
						avatar_hash={info.row.original.avatar_hash}
						personaname={info.row.original.personaname}
					/>
				),
			}),
			columnHelper.accessor("created_on", {
				header: "Updated",
				size: 120,
				cell: (info) => (
					<TableCell>
						<Typography>{renderDate(info.getValue())}</Typography>
					</TableCell>
				),
			}),
			columnHelper.accessor("updated_on", {
				header: "Updated",
				size: 120,
				cell: (info) => (
					<TableCell>
						<Typography>{renderDate(info.getValue())}</Typography>
					</TableCell>
				),
			}),
			columnHelper.display({
				id: "actions",
				size: 100,
				header: "Actions",
				cell: (info) => (
					<ButtonGroup variant={"contained"} fullWidth>
						<Button
							fullWidth
							startIcon={<DeleteIcon />}
							onClick={async () => {
								await onDelete(info.row.original);
							}}
						>
							Delete
						</Button>
					</ButtonGroup>
				),
			}),
		],
		[columnHelper, onDelete],
	);

	const table = useReactTable({
		data: whitelist,
		columns: columns,
		getCoreRowModel: getCoreRowModel(),
		manualPagination: true,
		autoResetPageIndex: true,
	});

	return <DataTable table={table} isLoading={isLoading} />;
};
