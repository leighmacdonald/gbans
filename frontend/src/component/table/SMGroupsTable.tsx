import NiceModal from "@ebay/nice-modal-react";
import AssuredWorkloadIcon from "@mui/icons-material/AssuredWorkload";
import DeleteIcon from "@mui/icons-material/Delete";
import EditIcon from "@mui/icons-material/Edit";
import GroupAddIcon from "@mui/icons-material/GroupAdd";
import IconButton from "@mui/material/IconButton";
import Tooltip from "@mui/material/Tooltip";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { createMRTColumnHelper, useMaterialReactTable } from "material-react-table";
import { useCallback, useMemo } from "react";
import { apiDeleteSMGroup, apiGetSMGroups } from "../../api";
import { useUserFlashCtx } from "../../hooks/useUserFlashCtx";
import type { SMAdmin, SMGroups } from "../../schema/sourcemod.ts";
import { logErr } from "../../util/errors";
import { renderDateTime } from "../../util/time.ts";
import { ConfirmationModal } from "../modal/ConfirmationModal.tsx";
import { SMGroupEditorModal } from "../modal/SMGroupEditorModal.tsx";
import { SMGroupOverridesModal } from "../modal/SMGroupOverridesModal.tsx";
import { createDefaultTableOptions } from "./options.ts";
import { SortableTable } from "./SortableTable.tsx";
import { TableCellString } from "./TableCellString";

const columnHelper = createMRTColumnHelper<SMGroups>();
const defaultOptions = createDefaultTableOptions<SMGroups>();

export const SMGroupsTable = () => {
	const { sendFlash, sendError } = useUserFlashCtx();
	const queryClient = useQueryClient();

	const { data, isLoading, isError } = useQuery({
		queryKey: ["serverGroups"],
		queryFn: async ({ signal }) => {
			return await apiGetSMGroups(signal);
		},
	});

	const onCreateGroup = useCallback(async () => {
		try {
			const group = (await NiceModal.show(SMGroupEditorModal)) as SMGroups;
			queryClient.setQueryData(["serverGroups"], [...(data ?? []), group]);
			sendFlash("success", `Group created successfully: ${group.name}`);
		} catch (e) {
			logErr(e);
			sendFlash("error", "Error trying to add group");
		}
	}, [data, queryClient, sendFlash]);

	const deleteGroupMutation = useMutation({
		mutationKey: ["SMGroupDelete"],
		mutationFn: async (group: SMGroups) => {
			const ac = new AbortController();
			await apiDeleteSMGroup(group.group_id, ac.signal);
			return group;
		},
		onSuccess: (group) => {
			queryClient.setQueryData(
				["serverGroups"],
				(data ?? []).filter((g) => g.group_id !== group.group_id),
			);
			sendFlash("success", "Group deleted successfully");
		},
		onError: sendError,
	});

	const onOverride = async (group: SMGroups) => {
		(await NiceModal.show(SMGroupOverridesModal, { group })) as SMAdmin;
	};

	const onDeleteGroup = async (group: SMGroups) => {
		try {
			const confirmed = (await NiceModal.show(ConfirmationModal, {
				title: "Delete group?",
				children: "This cannot be undone",
			})) as boolean;
			if (!confirmed) {
				return;
			}
			deleteGroupMutation.mutate(group);
		} catch (e) {
			sendFlash("error", `Failed to create confirmation modal: ${e}`);
		}
	};

	const onEditGroup = useCallback(
		async (group: SMGroups) => {
			try {
				const editedGroup = (await NiceModal.show(SMGroupEditorModal, {
					group,
				})) as SMGroups;
				queryClient.setQueryData(
					["serverGroups"],
					(data ?? []).map((g) => {
						return g.group_id !== editedGroup.group_id ? g : editedGroup;
					}),
				);
				sendFlash("success", `Group created successfully: ${group.name}`);
			} catch (e) {
				logErr(e);
				sendFlash("error", "Error trying to add group");
			}
		},
		[data, sendFlash, queryClient],
	);

	const columns = useMemo(
		() => [
			columnHelper.accessor("name", {
				header: "Name",
				grow: true,
				Cell: ({ cell }) => <TableCellString>{cell.getValue()}</TableCellString>,
			}),
			columnHelper.accessor("flags", {
				header: "Flags",
				grow: false,
				Cell: ({ cell }) => <TableCellString>{cell.getValue()}</TableCellString>,
			}),
			columnHelper.accessor("immunity_level", {
				header: "Immunity",
				grow: false,
				filterVariant: "range-slider",
				Cell: ({ cell }) => <TableCellString>{cell.getValue()}</TableCellString>,
			}),
			columnHelper.accessor("created_on", {
				header: "Created On",
				grow: false,
				Cell: ({ cell }) => <TableCellString>{renderDateTime(cell.getValue())}</TableCellString>,
			}),
			columnHelper.accessor("updated_on", {
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
		data: data ?? [],
		enableFilters: true,
		enableRowActions: true,
		state: {
			isLoading: isLoading,
			showAlertBanner: isError,
		},
		renderRowActionMenuItems: ({ row }) => [
			<Tooltip title={"Edit group overrides"} key={"edit"}>
				<IconButton
					color={"secondary"}
					onClick={async () => {
						await onOverride(row.original);
					}}
				>
					<AssuredWorkloadIcon />
				</IconButton>
			</Tooltip>,
			<IconButton
				key={"edit"}
				color={"warning"}
				onClick={async () => {
					await onEditGroup(row.original);
				}}
			>
				<EditIcon />
			</IconButton>,
			<IconButton
				key={"delete"}
				color={"error"}
				onClick={async () => {
					await onDeleteGroup(row.original);
				}}
			>
				<DeleteIcon />
			</IconButton>,
		],
		initialState: {
			...defaultOptions.initialState,
			sorting: [{ id: "name", desc: false }],
			columnVisibility: {
				name: true,
				identity: true,
			},
		},
	});

	return (
		<SortableTable
			table={table}
			title={"Groups"}
			buttons={[
				<IconButton onClick={onCreateGroup} key={"create"} sx={{ color: "primary.contrastText" }}>
					<GroupAddIcon />
				</IconButton>,
			]}
		/>
	);
};
