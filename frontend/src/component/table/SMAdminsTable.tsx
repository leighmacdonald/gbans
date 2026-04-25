import { useMutation, useQuery } from "@connectrpc/connect-query";
import NiceModal from "@ebay/nice-modal-react";
import DeleteIcon from "@mui/icons-material/Delete";
import EditIcon from "@mui/icons-material/Edit";
import GroupAddIcon from "@mui/icons-material/GroupAdd";
import GroupRemoveIcon from "@mui/icons-material/GroupRemove";
import PersonAddIcon from "@mui/icons-material/PersonAdd";
import IconButton from "@mui/material/IconButton";
import Tooltip from "@mui/material/Tooltip";
import { useQueryClient } from "@tanstack/react-query";
import { createMRTColumnHelper, useMaterialReactTable } from "material-react-table";
import { useCallback, useMemo } from "react";
import { useUserFlashCtx } from "../../hooks/useUserFlashCtx.ts";
import type { SMGroup, SMUser, SMUserValid } from "../../rpc/sourcemod/v1/sourcemod_pb.ts";
import {
	addAdminGroup,
	deleteAdmin,
	deleteAdminGroup,
	sMGroups,
	sMUsers,
} from "../../rpc/sourcemod/v1/sourcemod-SourcemodService_connectquery.ts";
import { ConfirmationModal } from "../modal/ConfirmationModal.tsx";
import { SMAdminEditorModal } from "../modal/SMAdminEditorModal.tsx";
import { SMGroupSelectModal } from "../modal/SMGroupSelectModal.tsx";
import { createDefaultTableOptions } from "./options.ts";
import { SortableTable } from "./SortableTable.tsx";
import { TableCellString } from "./TableCellString.tsx";

const columnHelperAdmins = createMRTColumnHelper<SMUser>();
const defaultOptionsAdmins = createDefaultTableOptions<SMUser>();

export const SMAdminsTable = () => {
	const { sendFlash, sendError } = useUserFlashCtx();
	const queryClient = useQueryClient();

	const { data: groups, isLoading: isLoadingGroups, isError: isErrorGroups } = useQuery(sMGroups);

	const { data: admins, isLoading: isLoadingAdmins, isError: isErrorAdmins } = useQuery(sMUsers);

	const onCreateAdmin = async () => {
		try {
			const admin = (await NiceModal.show(SMAdminEditorModal, {
				groups: groups?.groups,
			})) as SMUser;
			queryClient.setQueryData(["serverAdmins"], [...(admins?.users ?? []), admin]);
			sendFlash("success", `Admin created successfully: ${admin.name}`);
		} catch (e) {
			sendError(e);
		}
	};

	const deleteAdminFn = useMutation(deleteAdmin, {
		onSuccess: (_, req) => {
			queryClient.setQueryData(
				["serverAdmins"],
				(admins?.users ?? []).filter((a) => a.id !== req.adminId),
			);
			sendFlash("success", "Admin deleted successfully");
		},
		onError: sendError,
	});

	const addGroupMutation = useMutation(addAdminGroup, {
		onSuccess: (edited) => {
			queryClient.setQueryData(
				["serverAdmins"],
				(admins?.users ?? []).map((a) => {
					return a.id === edited.admin?.adminId ? edited : a;
				}),
			);
			sendFlash("success", `Admin updated successfully: ${edited.admin?.name}`);
		},
		onError: sendError,
	});

	const delGroupMutation = useMutation(deleteAdminGroup, {
		onSuccess: (_, req) => {
			// FIXME
			queryClient.setQueryData(
				["serverAdmins"],
				(admins?.users ?? []).filter((a) => {
					return a.id !== req.adminId;
				}),
			);
		},
		onError: sendError,
	});

	const onEdit = useCallback(
		async (admin: SMUser) => {
			try {
				const edited = (await NiceModal.show(SMAdminEditorModal, {
					admin: admin,
					groups: groups?.groups,
				})) as SMUser;
				queryClient.setQueryData(
					["serverAdmins"],
					(admins?.users ?? []).map((a) => {
						return a.id === edited.id ? edited : a;
					}),
				);
				sendFlash("success", `Admin updated successfully: ${admin.name}`);
			} catch (e) {
				sendError(e);
			}
		},
		[admins, groups, queryClient, sendError, sendFlash],
	);

	const onDelete = useCallback(
		async (admin: SMUser) => {
			try {
				const confirmed = (await NiceModal.show(ConfirmationModal, {
					title: "Delete admin?",
					children: "This cannot be undone",
				})) as boolean;
				if (!confirmed) {
					return;
				}
				deleteAdminFn.mutate({ adminId: admin.id });
			} catch (e) {
				sendFlash("error", `Failed to create confirmation modal: ${e}`);
			}
		},
		[sendFlash, deleteAdmin, deleteAdminFn.mutate],
	);

	const onAddGroup = useCallback(
		async (admin: SMUser) => {
			try {
				const existingGroupIds = admin.groups.map((g) => g.group_id);
				const group = (await NiceModal.show(SMGroupSelectModal, {
					groups: groups?.groups?.filter((g) => !existingGroupIds.includes(g.group_id)),
				})) as SMGroup;
				addGroupMutation.mutate({ admin, group });
			} catch (e) {
				sendError(e);
			}
		},
		[addGroupMutation, groups, sendError],
	);

	const onDelGroup = useCallback(
		async (admin: SMUser) => {
			try {
				const existingGroupIds = admin.groups.map((g) => g.group_id);
				const group = (await NiceModal.show(SMGroupSelectModal, {
					groups: groups?.filter((g) => existingGroupIds.includes(g.group_id)),
				})) as SMUserValid;
				delGroupMutation.mutate({ admin, group });
			} catch (e) {
				sendError(e);
			}
		},
		[delGroupMutation, groups, sendError],
	);

	const columns = useMemo(() => {
		return [
			columnHelperAdmins.accessor("name", {
				header: "Name",
				grow: true,
				Cell: ({ cell }) => <TableCellString>{cell.getValue()}</TableCellString>,
			}),
			columnHelperAdmins.accessor("authType", {
				header: "Auth Type",
				grow: false,
				Cell: ({ cell }) => <TableCellString>{cell.getValue()}</TableCellString>,
			}),
			columnHelperAdmins.accessor("identity", {
				header: "Identity",
				grow: false,
				Cell: ({ cell }) => <TableCellString>{cell.getValue()}</TableCellString>,
			}),
			// columnHelperAdmins.accessor("steamId", {
			// 	header: "SteamID",
			// 	grow: false,
			// 	Cell: ({ cell }) => <TableCellString>{cell.getValue()}</TableCellString>,
			// }),
			columnHelperAdmins.accessor("password", {
				header: "Password",
				grow: false,
				Cell: ({ cell }) => <TableCellString>{cell.getValue()}</TableCellString>,
			}),
			columnHelperAdmins.accessor("flags", {
				header: "Flags",
				grow: false,
				Cell: ({ cell }) => <TableCellString>{cell.getValue()}</TableCellString>,
			}),
			columnHelperAdmins.accessor("immunity", {
				header: "Immunity",
				grow: false,
				filterVariant: "range-slider",
				Cell: ({ cell }) => <TableCellString>{cell.getValue()}</TableCellString>,
			}),
			// columnHelperAdmins.accessor("createdOn", {
			// 	header: "Created On",
			// 	grow: false,
			// 	Cell: ({ cell }) => <TableCellString>{renderTimestamp(cell.getValue())}</TableCellString>,
			// }),
			// columnHelperAdmins.accessor("updatedOn", {
			// 	header: "Updated On",
			// 	grow: false,
			// 	Cell: ({ cell }) => <TableCellString>{renderTimestamp(cell.getValue())}</TableCellString>,
			// }),
		];
	}, []);

	const table = useMaterialReactTable({
		...defaultOptionsAdmins,
		columns,
		data: admins?.users ?? [],
		enableFilters: true,
		enableRowActions: true,
		state: {
			isLoading: isLoadingAdmins || isLoadingGroups,
			showAlertBanner: isErrorAdmins || isErrorGroups,
		},
		renderRowActionMenuItems: ({ row }) => [
			<Tooltip title={"Add user to group"} key={"add-btn"}>
				<span>
					<IconButton
						disabled={row.original.groups.length === groups?.length}
						color={"success"}
						onClick={async () => {
							await onAddGroup(row.original);
						}}
					>
						<GroupAddIcon />
					</IconButton>
				</span>
			</Tooltip>,
			<Tooltip title={"Remove user from group"} key={"remove-btn"}>
				<span>
					<IconButton
						disabled={row.original.groups.length === 0}
						color={"error"}
						onClick={async () => {
							await onDelGroup(row.original);
						}}
					>
						<GroupRemoveIcon />
					</IconButton>
				</span>
			</Tooltip>,
			<Tooltip title={"Edit admin"} key={"edit-btn"}>
				<IconButton
					color={"warning"}
					onClick={async () => {
						await onEdit(row.original);
					}}
				>
					<EditIcon />
				</IconButton>
			</Tooltip>,
			<Tooltip title={"Remove admin"} key={"del-btn"}>
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
			...defaultOptionsAdmins.initialState,
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
		<SortableTable
			table={table}
			title={"Admins"}
			buttons={[
				<IconButton onClick={onCreateAdmin} key="create-btn" sx={{ color: "primary.contrastText" }}>
					<PersonAddIcon />
				</IconButton>,
			]}
		/>
	);
};
