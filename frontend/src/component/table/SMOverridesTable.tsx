import NiceModal from "@ebay/nice-modal-react";
import AssuredWorkloadIcon from "@mui/icons-material/AssuredWorkload";
import DeleteIcon from "@mui/icons-material/Delete";
import EditIcon from "@mui/icons-material/Edit";
import IconButton from "@mui/material/IconButton";
import Tooltip from "@mui/material/Tooltip";
import { useQueryClient } from "@tanstack/react-query";
import { createMRTColumnHelper, useMaterialReactTable } from "material-react-table";
import { useCallback, useMemo } from "react";
import { useUserFlashCtx } from "../../hooks/useUserFlashCtx";
import { logErr } from "../../util/errors";
import { renderTimestamp } from "../../util/time";
import { ConfirmationModal } from "../modal/ConfirmationModal.tsx";
import { SMOverrideEditorModal } from "../modal/SMOverrideEditorModal.tsx";
import { createDefaultTableOptions } from "./options.ts";
import { SortableTable } from "./SortableTable.tsx";
import { TableCellString } from "./TableCellString";
import { deleteOverrides, overrides } from "../../rpc/sourcemod/v1/sourcemod-SourcemodService_connectquery.ts";
import type { Override } from "../../rpc/sourcemod/v1/sourcemod_pb.ts";
import { useMutation, useQuery } from "@connectrpc/connect-query";

const overrideColumnHelper = createMRTColumnHelper<Override>();
const defaultOptions = createDefaultTableOptions<Override>();

export const SMOverridesTable = () => {
	const { sendFlash, sendError } = useUserFlashCtx();
	const queryClient = useQueryClient();

	const { data: overridesList, isLoading, isError } = useQuery(overrides);

	const onCreateOverride = useCallback(async () => {
		try {
			const override = (await NiceModal.show(SMOverrideEditorModal, {})) as Override;
			queryClient.setQueryData(["serverOverrides"], [...(overridesList?.overrides ?? []), override]);
			sendFlash("success", `Group created successfully: ${override.name}`);
		} catch (e) {
			logErr(e);
			sendFlash("error", "Error trying to add group");
		}
	}, [queryClient, overrides, sendFlash]);

	const delOverrideMutation = useMutation(deleteOverrides, {
		onSuccess: (_, deleted) => {
			queryClient.setQueryData(
				["serverOverrides"],
				(overridesList?.overrides ?? []).filter((o) => {
					return o.overrideId !== deleted.overrideId;
				}),
			);
			sendFlash("success", `Override deleted successfully: ${deleted.overrideId}`);
		},
		onError: sendError,
	});

	const onEdit = async (override: Override) => {
		try {
			const edited = (await NiceModal.show(SMOverrideEditorModal, { override })) as Override;
			queryClient.setQueryData(
				["serverOverrides"],
				(overridesList?.overrides ?? []).map((o) => {
					return o.overrideId === edited.overrideId ? edited : o;
				}),
			);
			sendFlash("success", `Admin updated successfully: ${override.name}`);
		} catch (e) {
			logErr(e);
			sendFlash("error", "Error trying to update admin");
		}
	};

	const onDelete = useCallback(
		async (override: Override) => {
			try {
				const confirmed = (await NiceModal.show(ConfirmationModal, {
					title: "Delete override?",
					children: "This cannot be undone",
				})) as boolean;
				if (!confirmed) {
					return;
				}
				delOverrideMutation.mutate({ overrideId: override.overrideId });
			} catch (e) {
				sendFlash("error", `Failed to create confirmation modal: ${e}`);
			}
		},
		[delOverrideMutation, sendFlash],
	);

	const columns = useMemo(
		() => [
			overrideColumnHelper.accessor("name", {
				header: "Name",
				grow: true,
				Cell: ({ cell }) => <TableCellString>{cell.getValue()}</TableCellString>,
			}),
			overrideColumnHelper.accessor("overrideType", {
				header: "Type",
				grow: false,
				Cell: ({ cell }) => <TableCellString>{cell.getValue()}</TableCellString>,
			}),
			overrideColumnHelper.accessor("flags", {
				header: "Flags",
				grow: false,
				Cell: ({ cell }) => <TableCellString>{cell.getValue()}</TableCellString>,
			}),
			overrideColumnHelper.accessor("createdOn", {
				header: "Created On",
				grow: false,
				Cell: ({ cell }) => <TableCellString>{renderTimestamp(cell.getValue())}</TableCellString>,
			}),
			overrideColumnHelper.accessor("updatedOn", {
				header: "Updated On",
				grow: false,
				Cell: ({ cell }) => <TableCellString>{renderTimestamp(cell.getValue())}</TableCellString>,
			}),
		],
		[],
	);

	const table = useMaterialReactTable({
		...defaultOptions,
		columns,
		data: overridesList?.overrides ?? [],
		enableFilters: true,
		enableRowActions: true,
		state: {
			isLoading: isLoading,
			showAlertBanner: isError,
		},
		renderRowActionMenuItems: ({ row }) => [
			<Tooltip title={"Edit Override"} key={"edit-override"}>
				<IconButton
					color={"warning"}
					onClick={async () => {
						await onEdit(row.original);
					}}
				>
					<EditIcon />
				</IconButton>
			</Tooltip>,
			<Tooltip title={"Delete override"} key={"delete-override"}>
				<IconButton
					color={"error"}
					onClick={async () => {
						await onDelete(row.original);
					}}
				>
					<DeleteIcon />
				</IconButton>
			</Tooltip>,
		],
		initialState: {
			...defaultOptions.initialState,
			sorting: [{ id: "updated_on", desc: true }],
			columnVisibility: {
				name: true,
				identity: true,
			},
		},
	});

	return (
		<SortableTable
			table={table}
			title={"Command Overrides"}
			buttons={[
				<IconButton onClick={onCreateOverride} key="create-override" sx={{ color: "primary.contrastText" }}>
					<AssuredWorkloadIcon />
				</IconButton>,
			]}
		/>
	);
};
