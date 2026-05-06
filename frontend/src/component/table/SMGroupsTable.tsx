import { useMutation, useQuery } from "@connectrpc/connect-query";
import NiceModal from "@ebay/nice-modal-react";
import AssuredWorkloadIcon from "@mui/icons-material/AssuredWorkload";
import DeleteIcon from "@mui/icons-material/Delete";
import EditIcon from "@mui/icons-material/Edit";
import GroupAddIcon from "@mui/icons-material/GroupAdd";
import IconButton from "@mui/material/IconButton";
import Tooltip from "@mui/material/Tooltip";
import { useQueryClient } from "@tanstack/react-query";
import { createMRTColumnHelper, useMaterialReactTable } from "material-react-table";
import { useCallback, useMemo } from "react";
import { useUserFlashCtx } from "../../hooks/useUserFlashCtx";
import { sMGroups } from "../../rpc/sourcemod/v1/plugin-PluginService_connectquery.ts";
import type { Group } from "../../rpc/sourcemod/v1/sourcemod_pb.ts";
import { deleteGroup } from "../../rpc/sourcemod/v1/sourcemod-SourcemodService_connectquery.ts";
import { logErr } from "../../util/errors";
import { renderTimestamp } from "../../util/time.ts";
import { ConfirmationModal } from "../modal/ConfirmationModal.tsx";
import { SMGroupEditorModal } from "../modal/SMGroupEditorModal.tsx";
import { SMGroupOverridesModal } from "../modal/SMGroupOverridesModal.tsx";
import { createDefaultTableOptions } from "./options.ts";
import { SortableTable } from "./SortableTable.tsx";
import { TableCellString } from "./TableCellString";

const columnHelper = createMRTColumnHelper<Group>();
const defaultOptions = createDefaultTableOptions<Group>();

export const SMGroupsTable = () => {
	const { sendFlash, sendError } = useUserFlashCtx();
	const queryClient = useQueryClient();

	const { data, isLoading, isError } = useQuery(sMGroups);

	const onCreateGroup = useCallback(async () => {
		try {
			const group = (await NiceModal.show(SMGroupEditorModal)) as Group;
			queryClient.setQueryData(["serverGroups"], [...(data?.groups ?? []), group]);
			sendFlash("success", `Group created successfully: ${group.name}`);
		} catch (e) {
			logErr(e);
			sendFlash("error", "Error trying to add group");
		}
	}, [data, queryClient, sendFlash]);

	const deleteGroupMutation = useMutation(deleteGroup, {
		onSuccess: (_, req) => {
			queryClient.setQueryData(
				["serverGroups"],
				(data?.groups ?? []).filter((g) => g.groupId !== req.groupId),
			);
			sendFlash("success", "Group deleted successfully");
		},
		onError: sendError,
	});

	const onOverride = async (group: Group) => {
		await NiceModal.show(SMGroupOverridesModal, { group });
	};

	const onDeleteGroup = async (group: Group) => {
		try {
			const confirmed = (await NiceModal.show(ConfirmationModal, {
				title: "Delete group?",
				children: "This cannot be undone",
			})) as boolean;
			if (!confirmed) {
				return;
			}
			deleteGroupMutation.mutate({ groupId: group.groupId });
		} catch (e) {
			sendFlash("error", `Failed to create confirmation modal: ${e}`);
		}
	};

	const onEditGroup = useCallback(
		async (group: Group) => {
			try {
				const editedGroup = (await NiceModal.show(SMGroupEditorModal, {
					group,
				})) as Group;
				queryClient.setQueryData(
					["serverGroups"],
					(data?.groups ?? []).map((g) => {
						return g.groupId !== editedGroup.groupId ? g : editedGroup;
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
			columnHelper.accessor("immunityLevel", {
				header: "Immunity",
				grow: false,
				filterVariant: "range-slider",
				Cell: ({ cell }) => <TableCellString>{cell.getValue()}</TableCellString>,
			}),
			columnHelper.accessor("createdOn", {
				header: "Created On",
				grow: false,
				Cell: ({ cell }) => <TableCellString>{renderTimestamp(cell.getValue())}</TableCellString>,
			}),
			columnHelper.accessor("updatedOn", {
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
		data: data?.groups ?? [],
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
