import NiceModal, { useModal } from "@ebay/nice-modal-react";
import AddIcon from "@mui/icons-material/Add";
import DeleteIcon from "@mui/icons-material/Delete";
import EditIcon from "@mui/icons-material/Edit";
import IconButton from "@mui/material/IconButton";
import TableCell from "@mui/material/TableCell";
import Typography from "@mui/material/Typography";
import { Grid } from "@mui/system";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { createMRTColumnHelper, useMaterialReactTable } from "material-react-table";
import { useCallback, useMemo } from "react";
import { apiDeleteCIDRBlockWhitelist, apiGetCIDRBlockListsIPWhitelist } from "../../api";
import { useUserFlashCtx } from "../../hooks/useUserFlashCtx";
import type { WhitelistIP } from "../../schema/network";
import { logErr } from "../../util/errors";
import { renderDate } from "../../util/time";
import { ConfirmationModal } from "../modal/ConfirmationModal";
import { IPWhitelistEditorModal } from "../modal/IPWhitelistEditorModal";
import { createDefaultTableOptions } from "./options";
import { SortableTable } from "./SortableTable";

const columnHelper = createMRTColumnHelper<WhitelistIP>();
const defaultOptions = createDefaultTableOptions<WhitelistIP>();

export const IPWhitelistTable = () => {
	const confirmModal = useModal(ConfirmationModal);
	const { sendFlash, sendError } = useUserFlashCtx();
	const queryClient = useQueryClient();
	const { data, isLoading, isError } = useQuery({
		queryKey: ["networkIPWhitelist"],
		queryFn: async () => {
			return await apiGetCIDRBlockListsIPWhitelist();
		},
	});

	const onEdit = useCallback(
		async (source?: WhitelistIP) => {
			try {
				const newSource = (await NiceModal.show(IPWhitelistEditorModal, {
					source,
				})) as WhitelistIP;

				queryClient.setQueryData(
					["networkBlockListSourcesAdd"],
					(data ?? []).map((src) => {
						return src.cidr_block_whitelist_id === newSource.cidr_block_whitelist_id ? newSource : src;
					}),
				);
				sendFlash("success", "IP whitelist added");
			} catch (e) {
				sendFlash("error", `Failed to delete ip whitelist: ${e}`);
			}
		},
		[data, queryClient, sendFlash],
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

	const onDelete = useCallback(
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

	const columns = useMemo(
		() => [
			columnHelper.accessor("cidr_block_whitelist_id", {
				header: "ID",
				grow: false,
				Cell: ({ cell }) => <Typography>{cell.getValue() as number}</Typography>,
			}),
			columnHelper.accessor("address", {
				header: "CIDR Address",
				grow: true,
				Cell: ({ cell }) => (
					<TableCell>
						<Typography>{cell.getValue()}</Typography>
					</TableCell>
				),
			}),
			columnHelper.accessor("created_on", {
				header: "Created On",
				grow: true,
				Cell: ({ cell }) => (
					<TableCell>
						<Typography>{renderDate(cell.getValue() as Date)}</Typography>
					</TableCell>
				),
			}),
			columnHelper.accessor("updated_on", {
				header: "Updated On",
				grow: false,
				Cell: ({ cell }) => (
					<TableCell>
						<Typography>{renderDate(cell.getValue() as Date)}</Typography>
					</TableCell>
				),
			}),
		],
		[],
	);

	const table = useMaterialReactTable({
		...defaultOptions,
		columns,
		data: data ?? [],
		enableFilters: true,
		enableHiding: true,
		enableFacetedValues: true,
		state: {
			isLoading,
			showAlertBanner: isError,
		},
		initialState: {
			...defaultOptions.initialState,
			sorting: [{ id: "ban_id", desc: true }],
			columnVisibility: {
				cidr_block_whitelist_id: false,
				address: true,
				created_on: true,
				updated_on: false,
			},
		},
		enableRowActions: true,
		renderRowActionMenuItems: ({ row }) => [
			<IconButton
				key={"edit"}
				color={"warning"}
				onClick={async () => {
					await onEdit(row.original);
				}}
			>
				<EditIcon />
			</IconButton>,
			<IconButton
				key={"delete"}
				color={"error"}
				onClick={async () => {
					await onDelete(row.original);
				}}
			>
				<DeleteIcon />
			</IconButton>,
		],
	});

	return (
		<Grid container spacing={2}>
			<Grid size={{ xs: 12 }}>
				<SortableTable
					table={table}
					title={"CIDR/IP Whitelist"}
					buttons={[
						<IconButton
							key={"add"}
							onClick={async () => {
								await onEdit();
							}}
							sx={{ color: "primary.contrastText" }}
						>
							<AddIcon />
						</IconButton>,
					]}
				/>
			</Grid>
		</Grid>
	);
};
