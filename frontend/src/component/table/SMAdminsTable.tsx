import NiceModal from "@ebay/nice-modal-react";
import DeleteIcon from "@mui/icons-material/Delete";
import EditIcon from "@mui/icons-material/Edit";
import GroupAddIcon from "@mui/icons-material/GroupAdd";
import GroupRemoveIcon from "@mui/icons-material/GroupRemove";
import PersonAddIcon from "@mui/icons-material/PersonAdd";
import IconButton from "@mui/material/IconButton";
import Tooltip from "@mui/material/Tooltip";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { createMRTColumnHelper, useMaterialReactTable } from "material-react-table";
import { useCallback, useMemo } from "react";
import { apiAddAdminToGroup, apiDelAdminFromGroup, apiDeleteSMAdmin, apiGetSMAdmins, apiGetSMGroups } from "../../api";
import { useUserFlashCtx } from "../../hooks/useUserFlashCtx.ts";
import type { SMAdmin, SMGroups } from "../../schema/sourcemod.ts";
import { renderDateTime } from "../../util/time.ts";
import { ConfirmationModal } from "../modal/ConfirmationModal.tsx";
import { SMAdminEditorModal } from "../modal/SMAdminEditorModal.tsx";
import { SMGroupSelectModal } from "../modal/SMGroupSelectModal.tsx";
import { createDefaultTableOptions } from "./options.ts";
import { SortableTable } from "./SortableTable.tsx";
import { TableCellString } from "./TableCellString.tsx";

const columnHelperAdmins = createMRTColumnHelper<SMAdmin>();
const defaultOptionsAdmins = createDefaultTableOptions<SMAdmin>();

export const SMAdminsTable = () => {
	const { sendFlash, sendError } = useUserFlashCtx();
	const queryClient = useQueryClient();

	const {
		data: groups,
		isLoading: isLoadingGroups,
		isError: isErrorGroups,
	} = useQuery({
		queryKey: ["serverGroups"],
		queryFn: async ({ signal }) => {
			return await apiGetSMGroups(signal);
		},
	});

	const {
		data: admins,
		isLoading: isLoadingAdmins,
		isError: isErrorAdmins,
	} = useQuery({
		queryKey: ["serverAdmins"],
		queryFn: async ({ signal }) => {
			return await apiGetSMAdmins(signal);
		},
	});

	const onCreateAdmin = async () => {
		try {
			const admin = (await NiceModal.show(SMAdminEditorModal, {
				groups,
			})) as SMAdmin;
			queryClient.setQueryData(["serverAdmins"], [...(admins ?? []), admin]);
			sendFlash("success", `Admin created successfully: ${admin.name}`);
		} catch (e) {
			sendError(e);
		}
	};

	const deleteAdmin = useMutation({
		mutationKey: ["SMAdminDelete"],
		mutationFn: async (admin: SMAdmin) => {
			const ac = new AbortController();
			await apiDeleteSMAdmin(admin.admin_id, ac.signal);
			return admin;
		},
		onSuccess: (admin) => {
			queryClient.setQueryData(
				["serverAdmins"],
				(admins ?? []).filter((a) => a.admin_id !== admin.admin_id),
			);
			sendFlash("success", "Admin deleted successfully");
		},
		onError: sendError,
	});

	const addGroupMutation = useMutation({
		mutationKey: ["addAdminGroup"],
		mutationFn: async ({ admin, group }: { admin: SMAdmin; group: SMGroups }) => {
			const ac = new AbortController();
			return await apiAddAdminToGroup(admin.admin_id, group.group_id, ac.signal);
		},
		onSuccess: (edited) => {
			queryClient.setQueryData(
				["serverAdmins"],
				(admins ?? []).map((a) => {
					return a.admin_id === edited.admin_id ? edited : a;
				}),
			);
			sendFlash("success", `Admin updated successfully: ${edited.name}`);
		},
		onError: sendError,
	});

	const delGroupMutation = useMutation({
		mutationKey: ["addAdminGroup"],
		mutationFn: async ({ admin, group }: { admin: SMAdmin; group: SMGroups }) => {
			const ac = new AbortController();
			return await apiDelAdminFromGroup(admin.admin_id, group.group_id, ac.signal);
		},
		onSuccess: (edited) => {
			// FIXME
			queryClient.setQueryData(
				["serverAdmins"],
				(admins ?? []).filter((a) => {
					return a.admin_id !== edited.admin_id;
				}),
			);
			sendFlash("success", `Admin updated successfully: ${edited.name}`);
		},
		onError: sendError,
	});

	const onEdit = useCallback(
		async (admin: SMAdmin) => {
			try {
				const edited = (await NiceModal.show(SMAdminEditorModal, {
					admin,
					groups,
				})) as SMAdmin;
				queryClient.setQueryData(
					["serverAdmins"],
					(admins ?? []).map((a) => {
						return a.admin_id === edited.admin_id ? edited : a;
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
		async (admin: SMAdmin) => {
			try {
				const confirmed = (await NiceModal.show(ConfirmationModal, {
					title: "Delete admin?",
					children: "This cannot be undone",
				})) as boolean;
				if (!confirmed) {
					return;
				}
				deleteAdmin.mutate(admin);
			} catch (e) {
				sendFlash("error", `Failed to create confirmation modal: ${e}`);
			}
		},
		[sendFlash, deleteAdmin],
	);

	const onAddGroup = useCallback(
		async (admin: SMAdmin) => {
			try {
				const existingGroupIds = admin.groups.map((g) => g.group_id);
				const group = (await NiceModal.show(SMGroupSelectModal, {
					groups: groups?.filter((g) => !existingGroupIds.includes(g.group_id)),
				})) as SMGroups;
				addGroupMutation.mutate({ admin, group });
			} catch (e) {
				sendError(e);
			}
		},
		[addGroupMutation, groups, sendError],
	);

	const onDelGroup = useCallback(
		async (admin: SMAdmin) => {
			try {
				const existingGroupIds = admin.groups.map((g) => g.group_id);
				const group = (await NiceModal.show(SMGroupSelectModal, {
					groups: groups?.filter((g) => existingGroupIds.includes(g.group_id)),
				})) as SMGroups;
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
			columnHelperAdmins.accessor("auth_type", {
				header: "Auth Type",
				grow: false,
				Cell: ({ cell }) => <TableCellString>{cell.getValue()}</TableCellString>,
			}),
			columnHelperAdmins.accessor("identity", {
				header: "Identity",
				grow: false,
				Cell: ({ cell }) => <TableCellString>{cell.getValue()}</TableCellString>,
			}),
			columnHelperAdmins.accessor("steam_id", {
				header: "SteamID",
				grow: false,
				Cell: ({ cell }) => <TableCellString>{cell.getValue()}</TableCellString>,
			}),
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
			columnHelperAdmins.accessor("created_on", {
				header: "Created On",
				grow: false,
				Cell: ({ cell }) => <TableCellString>{renderDateTime(cell.getValue())}</TableCellString>,
			}),
			columnHelperAdmins.accessor("updated_on", {
				header: "Updated On",
				grow: false,
				Cell: ({ cell }) => <TableCellString>{renderDateTime(cell.getValue())}</TableCellString>,
			}),
		];
	}, []);

	const table = useMaterialReactTable({
		...defaultOptionsAdmins,
		columns,
		data: admins ?? [],
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
