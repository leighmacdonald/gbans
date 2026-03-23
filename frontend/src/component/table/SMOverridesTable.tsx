import NiceModal from "@ebay/nice-modal-react";
import AssuredWorkloadIcon from "@mui/icons-material/AssuredWorkload";
import DeleteIcon from "@mui/icons-material/Delete";
import EditIcon from "@mui/icons-material/Edit";
import IconButton from "@mui/material/IconButton";
import Tooltip from "@mui/material/Tooltip";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { createMRTColumnHelper, useMaterialReactTable } from "material-react-table";
import { useCallback, useMemo } from "react";
import { apiDeleteSMOverride, apiGetSMOverrides } from "../../api";
import { useUserFlashCtx } from "../../hooks/useUserFlashCtx";
import type { SMOverrides } from "../../schema/sourcemod.ts";
import { logErr } from "../../util/errors";
import { renderDateTime } from "../../util/time";
import { ConfirmationModal } from "../modal/ConfirmationModal.tsx";
import { SMOverrideEditorModal } from "../modal/SMOverrideEditorModal.tsx";
import { createDefaultTableOptions } from "./options.ts";
import { SortableTable } from "./SortableTable.tsx";
import { TableCellString } from "./TableCellString";

const overrideColumnHelper = createMRTColumnHelper<SMOverrides>();
const defaultOptions = createDefaultTableOptions<SMOverrides>();

export const SMOverridesTable = () => {
	const { sendFlash, sendError } = useUserFlashCtx();
	const queryClient = useQueryClient();

	const {
		data: overrides,
		isLoading,
		isError,
	} = useQuery({
		queryKey: ["serverOverrides"],
		queryFn: async ({ signal }) => {
			return await apiGetSMOverrides(signal);
		},
	});

	const onCreateOverride = useCallback(async () => {
		try {
			const override = (await NiceModal.show(SMOverrideEditorModal, {})) as SMOverrides;
			queryClient.setQueryData(["serverOverrides"], [...(overrides ?? []), override]);
			sendFlash("success", `Group created successfully: ${override.name}`);
		} catch (e) {
			logErr(e);
			sendFlash("error", "Error trying to add group");
		}
	}, [queryClient, overrides, sendFlash]);

	const delOverrideMutation = useMutation({
		mutationKey: ["delOverride"],
		mutationFn: async ({ override }: { override: SMOverrides }) => {
			const ac = new AbortController();
			await apiDeleteSMOverride(override.override_id, ac.signal);
			return override;
		},
		onSuccess: (deleted) => {
			queryClient.setQueryData(
				["serverOverrides"],
				(overrides ?? []).filter((o) => {
					return o.override_id !== deleted.override_id;
				}),
			);
			sendFlash("success", `Override deleted successfully: ${deleted.name}`);
		},
		onError: sendError,
	});

	const onEdit = async (override: SMOverrides) => {
		try {
			const edited = (await NiceModal.show(SMOverrideEditorModal, { override })) as SMOverrides;
			queryClient.setQueryData(
				["serverOverrides"],
				(overrides ?? []).map((o) => {
					return o.override_id === edited.override_id ? edited : o;
				}),
			);
			sendFlash("success", `Admin updated successfully: ${override.name}`);
		} catch (e) {
			logErr(e);
			sendFlash("error", "Error trying to update admin");
		}
	};

	const onDelete = useCallback(
		async (override: SMOverrides) => {
			try {
				const confirmed = (await NiceModal.show(ConfirmationModal, {
					title: "Delete override?",
					children: "This cannot be undone",
				})) as boolean;
				if (!confirmed) {
					return;
				}
				delOverrideMutation.mutate({ override });
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
			overrideColumnHelper.accessor("type", {
				header: "Type",
				grow: false,
				Cell: ({ cell }) => <TableCellString>{cell.getValue()}</TableCellString>,
			}),
			overrideColumnHelper.accessor("flags", {
				header: "Flags",
				grow: false,
				Cell: ({ cell }) => <TableCellString>{cell.getValue()}</TableCellString>,
			}),
			overrideColumnHelper.accessor("created_on", {
				header: "Created On",
				grow: false,
				Cell: ({ cell }) => <TableCellString>{renderDateTime(cell.getValue())}</TableCellString>,
			}),
			overrideColumnHelper.accessor("updated_on", {
				header: "Updated On",
				grow: false,
				Cell: ({ cell }) => <TableCellString>{renderDateTime(cell.getValue())}</TableCellString>,
			}),
		],
		[],
	);

	const table = useMaterialReactTable({
		...defaultOptions,
		columns,
		data: overrides ?? [],
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
