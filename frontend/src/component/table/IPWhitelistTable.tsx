import NiceModal, { useModal } from "@ebay/nice-modal-react";
import AddIcon from "@mui/icons-material/Add";
import DeleteIcon from "@mui/icons-material/Delete";
import EditIcon from "@mui/icons-material/Edit";
import { useTheme } from "@mui/material";
import IconButton from "@mui/material/IconButton";
import { Grid } from "@mui/system";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { createMRTColumnHelper, useMaterialReactTable } from "material-react-table";
import { useCallback, useMemo } from "react";
import { apiDeleteCIDRBlockWhitelist, apiGetCIDRBlockListsIPWhitelist } from "../../api";
import { useUserFlashCtx } from "../../hooks/useUserFlashCtx";
import type { WhitelistIP } from "../../schema/network";
import { logErr } from "../../util/errors";
import { cidrHostCount } from "../../util/strings";
import { renderDateTime } from "../../util/time";
import { ConfirmationModal } from "../modal/ConfirmationModal";
import { IPWhitelistEditorModal } from "../modal/IPWhitelistEditorModal";
import RouterLink from "../RouterLink";
import { createDefaultTableOptions, setColumnFilter } from "./options";
import { SortableTable } from "./SortableTable";

const columnHelper = createMRTColumnHelper<WhitelistIP>();
const defaultOptions = createDefaultTableOptions<WhitelistIP>();

export const IPWhitelistTable = () => {
	const confirmModal = useModal(ConfirmationModal);
	const { sendFlash, sendError } = useUserFlashCtx();
	const queryClient = useQueryClient();
	const theme = useTheme();

	const { data, isLoading, isError } = useQuery({
		queryKey: ["networkIPWhitelist"],
		queryFn: async ({ signal }) => {
			return await apiGetCIDRBlockListsIPWhitelist(signal);
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
			const ac = new AbortController();
			await apiDeleteCIDRBlockWhitelist(variables.cidr_block_whitelist_id, ac.signal);
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
			}),
			columnHelper.accessor("address", {
				header: "CIDR Address",
				grow: true,
				Cell: ({ cell }) => (
					<RouterLink
						style={{ color: theme.palette.primary.light }}
						to={"/admin/network/playersbyip"}
						search={setColumnFilter({}, "ip_addr", cell.getValue())}
					>
						{cell.getValue()}
					</RouterLink>
				),
			}),
			columnHelper.display({
				id: "hosts",
				header: "Hosts",
				Cell: ({ row }) => cidrHostCount(row.original.address),
			}),
			columnHelper.accessor("created_on", {
				header: "Created On",
				grow: true,
				Cell: ({ cell }) => renderDateTime(cell.getValue()),
			}),
			columnHelper.accessor("updated_on", {
				header: "Updated On",
				grow: false,
				Cell: ({ cell }) => renderDateTime(cell.getValue()),
			}),
		],
		[theme],
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
				hosts: true,
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
