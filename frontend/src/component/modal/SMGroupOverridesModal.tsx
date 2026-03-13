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
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { useCallback, useMemo } from "react";
import "video-react/dist/video-react.css";
import { createMRTColumnHelper, useMaterialReactTable } from "material-react-table";
import { apiDeleteSMGroupOverride, apiGetSMGroupOverrides } from "../../api";
import { useUserFlashCtx } from "../../hooks/useUserFlashCtx.ts";
import type { SMGroupOverrides, SMGroups } from "../../schema/sourcemod.ts";
import { logErr } from "../../util/errors.ts";
import { renderDateTime } from "../../util/time.ts";
import { Heading } from "../Heading";
import { createDefaultTableOptions } from "../table/options.ts";
import { SortableTable } from "../table/SortableTable.tsx";
import { TableCellString } from "../table/TableCellString.tsx";
import { ConfirmationModal } from "./ConfirmationModal.tsx";
import { SMGroupOverrideEditorModal } from "./SMGroupOverrideEditorModal.tsx";

const columnHelper = createMRTColumnHelper<SMGroupOverrides>();
const defaultOptions = createDefaultTableOptions<SMGroupOverrides>();

export const SMGroupOverridesModal = NiceModal.create(({ group }: { group: SMGroups }) => {
	const modal = useModal();
	const queryClient = useQueryClient();
	const { sendFlash, sendError } = useUserFlashCtx();

	const { data, isLoading, isError } = useQuery({
		queryKey: ["serverGroupOverrides", { group_id: group.group_id }],
		queryFn: async () => {
			return await apiGetSMGroupOverrides(group.group_id);
		},
	});

	const onCreate = useCallback(async () => {
		try {
			const created = (await NiceModal.show(SMGroupOverrideEditorModal, { group })) as SMGroupOverrides;
			queryClient.setQueryData(
				["serverGroupOverrides", { group_id: group.group_id }],
				[...(data ?? []), created],
			);
			sendFlash("success", `Group override created successfully: ${created.name}`);
		} catch (e) {
			logErr(e);
			sendFlash("error", "Error trying to add group override");
		}
	}, [data, queryClient, sendFlash, group]);

	const delOverrideMutation = useMutation({
		mutationKey: ["deleteGroupOverride"],
		mutationFn: async ({ groupOverride }: { groupOverride: SMGroupOverrides }) => {
			await apiDeleteSMGroupOverride(groupOverride.group_override_id);
			return groupOverride;
		},
		onSuccess: (edited) => {
			queryClient.setQueryData(
				["serverGroupOverrides", { group_id: edited.group_id }],
				(data ?? []).filter((o) => {
					return o.group_override_id !== edited.group_override_id;
				}),
			);
			sendFlash("success", `Group override deleted successfully: ${edited.name}`);
		},
		onError: sendError,
	});

	const onEdit = useCallback(
		async (override: SMGroupOverrides) => {
			try {
				const edited = (await NiceModal.show(SMGroupOverrideEditorModal, { override })) as SMGroupOverrides;
				queryClient.setQueryData(
					["serverGroupOverrides", { group_id: group.group_id }],
					(data ?? []).map((o) => {
						return o.group_override_id === edited.group_override_id ? edited : o;
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
		async (groupOverride: SMGroupOverrides) => {
			try {
				const confirmed = await NiceModal.show(ConfirmationModal, {
					title: "Delete override?",
					children: "This cannot be undone",
				});
				if (!confirmed) {
					return;
				}
				delOverrideMutation.mutate({ groupOverride });
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
			columnHelper.accessor("type", {
				header: "Type",
				grow: false,
				Cell: ({ cell }) => <TableCellString>{cell.getValue()}</TableCellString>,
			}),
			columnHelper.accessor("access", {
				header: "Access",
				grow: false,
				Cell: ({ cell }) => <TableCellString>{cell.getValue()}</TableCellString>,
			}),
			columnHelper.accessor("created_on", {
				header: "Created",
				grow: false,
				Cell: ({ cell }) => <TableCellString>{renderDateTime(cell.getValue())}</TableCellString>,
			}),
			columnHelper.accessor("updated_on", {
				header: "Updated",
				grow: false,
				Cell: ({ cell }) => <TableCellString>{renderDateTime(cell.getValue())}</TableCellString>,
			}),
		],
		[],
	);

	const table = useMaterialReactTable({
		...defaultOptions,
		columns,
		data: data ?? [],
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
