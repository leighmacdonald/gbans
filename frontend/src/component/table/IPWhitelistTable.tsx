import { useMutation, useQuery } from "@connectrpc/connect-query";
import NiceModal, { useModal } from "@ebay/nice-modal-react";
import AddIcon from "@mui/icons-material/Add";
import DeleteIcon from "@mui/icons-material/Delete";
import EditIcon from "@mui/icons-material/Edit";
import { useTheme } from "@mui/material";
import IconButton from "@mui/material/IconButton";
import { Grid } from "@mui/system";
import { useQueryClient } from "@tanstack/react-query";
import { createMRTColumnHelper, useMaterialReactTable } from "material-react-table";
import { useCallback, useMemo } from "react";
import { useUserFlashCtx } from "../../hooks/useUserFlashCtx";
import type { CIDRBlockWhitelist } from "../../rpc/network/v1/blocklist_pb.ts";
import {
	whitelistAddress,
	whitelistAddressDelete,
} from "../../rpc/network/v1/blocklist-BlocklistService_connectquery.ts";
import { logErr } from "../../util/errors";
import { cidrHostCount } from "../../util/strings";
import { renderTimestamp } from "../../util/time";
import { ConfirmationModal } from "../modal/ConfirmationModal";
import { IPWhitelistEditorModal } from "../modal/IPWhitelistEditorModal";
import RouterLink from "../RouterLink";
import { createDefaultTableOptions, setColumnFilter } from "./options";
import { SortableTable } from "./SortableTable";

const columnHelper = createMRTColumnHelper<CIDRBlockWhitelist>();
const defaultOptions = createDefaultTableOptions<CIDRBlockWhitelist>();

export const IPWhitelistTable = () => {
	const confirmModal = useModal(ConfirmationModal);
	const { sendFlash, sendError } = useUserFlashCtx();
	const queryClient = useQueryClient();
	const theme = useTheme();

	const { data, isLoading, isError } = useQuery(whitelistAddress);

	const onEdit = useCallback(
		async (source?: CIDRBlockWhitelist) => {
			try {
				const newSource = (await NiceModal.show(IPWhitelistEditorModal, {
					source,
				})) as CIDRBlockWhitelist;

				queryClient.setQueryData(
					["networkBlockListSourcesAdd"],
					(data?.whitelisted ?? []).map((src) => {
						return src.cidrBlockWhitelistId === newSource.cidrBlockWhitelistId ? newSource : src;
					}),
				);
				sendFlash("success", "IP whitelist added");
			} catch (e) {
				sendFlash("error", `Failed to delete ip whitelist: ${e}`);
			}
		},
		[data, queryClient, sendFlash],
	);

	const ipWhitelistMutation = useMutation(whitelistAddressDelete, {
		onSuccess: () => {
			sendFlash("success", "IP whitelist deleted");
		},
		onError: sendError,
	});

	const onDelete = useCallback(
		async (source: CIDRBlockWhitelist) => {
			try {
				const confirmed = await confirmModal.show({
					title: "Delete CIDR Whitelist?",
					children: "This action is permanent",
				});
				if (confirmed) {
					ipWhitelistMutation.mutate({
						cidrBlockWhitelistId: source.cidrBlockWhitelistId,
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
			columnHelper.accessor("cidrBlockWhitelistId", {
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
			columnHelper.accessor("createdOn", {
				header: "Created On",
				grow: true,
				Cell: ({ cell }) => renderTimestamp(cell.getValue()),
			}),
			columnHelper.accessor("updatedOn", {
				header: "Updated On",
				grow: false,
				Cell: ({ cell }) => renderTimestamp(cell.getValue()),
			}),
		],
		[theme],
	);

	const table = useMaterialReactTable({
		...defaultOptions,
		columns,
		data: data?.whitelisted ?? [],
		enableFilters: true,
		enableHiding: true,
		enableFacetedValues: true,
		state: {
			isLoading,
			showAlertBanner: isError,
		},
		initialState: {
			...defaultOptions.initialState,
			sorting: [{ id: "cidrBlockWhitelistId", desc: true }],
			columnVisibility: {
				cidrBlockWhitelistId: false,
				address: true,
				hosts: true,
				createdOn: true,
				updatedOn: false,
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
