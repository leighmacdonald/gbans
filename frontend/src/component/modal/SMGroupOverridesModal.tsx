import { useMutation, useQuery } from "@connectrpc/connect-query";
import NiceModal, { muiDialogV5, useModal } from "@ebay/nice-modal-react";
import AddIcon from "@mui/icons-material/Add";
import CloseIcon from "@mui/icons-material/Close";
import DeleteIcon from "@mui/icons-material/Delete";
import EditIcon from "@mui/icons-material/Edit";
import GroupsIcon from "@mui/icons-material/Groups";
import { Dialog, DialogActions, DialogContent, DialogTitle } from "@mui/material";
import Button from "@mui/material/Button";
import ButtonGroup from "@mui/material/ButtonGroup";
import Grid from "@mui/material/Grid";
import IconButton from "@mui/material/IconButton";
import { useQueryClient } from "@tanstack/react-query";
import { createMRTColumnHelper, useMaterialReactTable } from "material-react-table";
import { useCallback, useMemo } from "react";
import { useUserFlashCtx } from "../../hooks/useUserFlashCtx.ts";
import type { Group, GroupOverrides } from "../../rpc/sourcemod/v1/sourcemod_pb.ts";
import { deleteGroupOverride, groupOverrides } from "../../rpc/sourcemod/v1/sourcemod-SourcemodService_connectquery.ts";
import { logErr } from "../../util/errors.ts";
import { renderTimestamp } from "../../util/time.ts";
import { Heading } from "../Heading";
import { createDefaultTableOptions } from "../table/options.ts";
import { SortableTable } from "../table/SortableTable.tsx";
import { TableCellString } from "../table/TableCellString.tsx";
import { ConfirmationModal } from "./ConfirmationModal.tsx";
import { SMGroupOverrideEditorModal } from "./SMGroupOverrideEditorModal.tsx";

const columnHelper = createMRTColumnHelper<GroupOverrides>();
const defaultOptions = createDefaultTableOptions<GroupOverrides>();

export const SMGroupOverridesModal = NiceModal.create(({ group }: { group: Group }) => {
	const modal = useModal();
	const queryClient = useQueryClient();
	const { sendFlash, sendError } = useUserFlashCtx();

	const { data, isLoading, isError } = useQuery(groupOverrides);

	const onCreate = useCallback(async () => {
		try {
			const created = (await NiceModal.show(SMGroupOverrideEditorModal, { group })) as GroupOverrides;
			queryClient.setQueryData(
				["serverGroupOverrides", { group_id: group.groupId }],
				[...(data?.overrides ?? []), created],
			);
			sendFlash("success", `Group override created successfully: ${created.name}`);
		} catch (e) {
			logErr(e);
			sendFlash("error", "Error trying to add group override");
		}
	}, [data, queryClient, sendFlash, group]);

	const delOverrideMutation = useMutation(deleteGroupOverride, {
		onSuccess: (_, req) => {
			queryClient.setQueryData(
				["serverGroupOverrides", { group_id: req.groupOverrideId }],
				(data?.overrides ?? []).filter((o) => {
					return o.groupOverrideId !== req.groupOverrideId;
				}),
			);
			sendFlash("success", `Group override deleted successfully: ${req.groupOverrideId}`);
		},
		onError: sendError,
	});

	const onEdit = useCallback(
		async (override: GroupOverrides) => {
			try {
				const edited = (await NiceModal.show(SMGroupOverrideEditorModal, { override })) as GroupOverrides;
				queryClient.setQueryData(
					["serverGroupOverrides", { group_id: group.groupId }],
					(data?.overrides ?? []).map((o) => {
						return o.groupOverrideId === edited.groupOverrideId ? edited : o;
					}),
				);
				sendFlash("success", `Group override updated successfully: ${override.name}`);
			} catch (e) {
				logErr(e);
				sendFlash("error", "Error trying to edit group override");
			}
		},
		[queryClient, sendFlash, group, data],
	);

	const onDelete = useCallback(
		async (groupOverride: GroupOverrides) => {
			try {
				const confirmed = await NiceModal.show(ConfirmationModal, {
					title: "Delete override?",
					children: "This cannot be undone",
				});
				if (!confirmed) {
					return;
				}
				delOverrideMutation.mutate({ groupOverrideId: groupOverride.groupOverrideId });
			} catch (e) {
				sendFlash("error", `Failed to create confirmation modal: ${e}`);
			}
		},
		[delOverrideMutation, sendFlash],
	);

	const columns = useMemo(
		() => [
			columnHelper.accessor("name", {
				header: "Name",
				grow: true,
				Cell: ({ cell }) => <TableCellString>{cell.getValue()}</TableCellString>,
			}),
			columnHelper.accessor("overrideType", {
				header: "Type",
				grow: false,
				Cell: ({ cell }) => <TableCellString>{cell.getValue()}</TableCellString>,
			}),
			columnHelper.accessor("overrideAccess", {
				header: "Access",
				grow: false,
				Cell: ({ cell }) => <TableCellString>{cell.getValue()}</TableCellString>,
			}),
			columnHelper.accessor("createdOn", {
				header: "Created",
				grow: false,
				Cell: ({ cell }) => <TableCellString>{renderTimestamp(cell.getValue())}</TableCellString>,
			}),
			columnHelper.accessor("updatedOn", {
				header: "Updated",
				grow: false,
				Cell: ({ cell }) => <TableCellString>{renderTimestamp(cell.getValue())}</TableCellString>,
			}),
		],
		[],
	);

	const table = useMaterialReactTable({
		...defaultOptions,
		columns,
		data: data?.overrides ?? [],
		enableFilters: true,
		enableRowActions: true,
		state: {
			isLoading,
			showAlertBanner: isError,
		},
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
		initialState: {
			...defaultOptions.initialState,
			sorting: [{ id: "updated_on", desc: true }],
			columnVisibility: {
				name: true,
				identity: true,
				created_on: false,
				updated_on: false,
				steam_id: false,
				password: false,
			},
		},
	});

	return (
		<Dialog fullWidth {...muiDialogV5(modal)}>
			<DialogTitle component={Heading} iconLeft={<GroupsIcon />}>
				Group Overrides
			</DialogTitle>

			<DialogContent>
				<SortableTable table={table} title={"Admins"} />
			</DialogContent>

			<DialogActions>
				<Grid container>
					<Grid size={{ xs: 12 }}>
						<ButtonGroup>
							<Button startIcon={<AddIcon />} color={"success"} onClick={onCreate}>
								New
							</Button>
							<Button key={"close-button"} onClick={modal.hide} color={"error"} startIcon={<CloseIcon />}>
								Close
							</Button>
						</ButtonGroup>
					</Grid>
				</Grid>
			</DialogActions>
		</Dialog>
	);
});
