import { createConnectQueryKey, useMutation, useQuery, useTransport } from "@connectrpc/connect-query";
import NiceModal from "@ebay/nice-modal-react";
import AddIcon from "@mui/icons-material/Add";
import EditIcon from "@mui/icons-material/Edit";
import GroupsIcon from "@mui/icons-material/Groups";
import RemoveIcon from "@mui/icons-material/Remove";
import { useTheme } from "@mui/material";
import Button from "@mui/material/Button";
import Grid from "@mui/material/Grid";
import IconButton from "@mui/material/IconButton";
import Tooltip from "@mui/material/Tooltip";
import Typography from "@mui/material/Typography";
import { useQueryClient } from "@tanstack/react-query";
import { createFileRoute } from "@tanstack/react-router";
import { createMRTColumnHelper, useMaterialReactTable } from "material-react-table";
import { useCallback, useMemo } from "react";
import { ConfirmationModal } from "../component/modal/ConfirmationModal";
import { RoleCreateModal } from "../component/modal/RoleCreateModal";
import { RoleEditModal } from "../component/modal/RoleEditModal";
import { RowActionContainer } from "../component/RowActionContainer";
import { createDefaultTableOptions, makeRowActionsDefOptions } from "../component/table/options";
import { RoleUsersTable } from "../component/table/RoleUsersTable";
import { SMImmunityTable } from "../component/table/SMImmunityTable";
import { SMOverridesTable } from "../component/table/SMOverridesTable";
import { SortableTable } from "../component/table/SortableTable";
import { useUserFlashCtx } from "../hooks/useUserFlashCtx";
import { Permission, type Role, RolesService } from "../rpc/roles/v1/roles_pb";
import { roleDelete, roleList } from "../rpc/roles/v1/roles-RolesService_connectquery";
import { logErr } from "../util/errors";
import { enumValues } from "../util/lists";
import { renderTimestampDateTime } from "../util/time";

const columnHelper = createMRTColumnHelper<Role>();
const defaultOptions = createDefaultTableOptions<Role>();

export const Route = createFileRoute("/_admin/admin/roles")({
	component: AdminRoles,
});

function AdminRoles() {
	const { sendFlash } = useUserFlashCtx();
	const queryClient = useQueryClient();
	const transport = useTransport();
	const theme = useTheme();

	const { data: rolesResponse, isError, isLoading } = useQuery(roleList, {});
	const columns = useMemo(() => {
		return [
			columnHelper.accessor("roleId", {
				header: "Role ID",
				grow: false,
				enableColumnFilter: false,
				size: 40,
			}),
			columnHelper.accessor("roleName", {
				header: "Role Name",
				grow: false,
				size: 150,
			}),
			columnHelper.accessor("userCount", {
				header: "Count",
				grow: false,
				enableColumnFilter: false,
				size: 100,
			}),
			columnHelper.accessor("permissions", {
				header: "Permissions",
				grow: true,
				enableColumnFilter: true,
				enableSorting: true,
				filterVariant: "multi-select",
				filterSelectOptions: enumValues(Permission, true).map((v) => ({ value: v, label: Permission[v] })),
				filterFn: (row, _, value) => {
					if (!value || value.length === 0) {
						return true;
					}
					const permissions = row.getValue("permissions") as number[];
					return (value as number[]).filter((s) => permissions.includes(s)).length > 0;
				},
				Cell: ({ cell }) => (
					<Typography>
						{cell
							.getValue()
							.map((p) => Permission[p])
							.join(", ")}
					</Typography>
				),
			}),
			columnHelper.accessor("createdOn", {
				header: "Created On",
				grow: false,
				size: 120,
				Cell: ({ cell }) => <Typography>{renderTimestampDateTime(cell.getValue())}</Typography>,
			}),
			columnHelper.accessor("updatedOn", {
				header: "Updated On",
				grow: false,
				size: 120,
				Cell: ({ cell }) => <Typography>{renderTimestampDateTime(cell.getValue())}</Typography>,
			}),
		];
	}, []);

	const resetRoles = useCallback(() => {
		queryClient.invalidateQueries({
			queryKey: createConnectQueryKey({
				schema: RolesService.method.roleList,
				cardinality: "finite",
				transport,
				input: {},
			}),
		});
	}, [queryClient, transport]);

	const onCreate = useCallback(async () => {
		try {
			const role = (await NiceModal.show(RoleCreateModal, {})) as Role;
			sendFlash("success", `Role created successfully: ${role.roleName}`);
			resetRoles();
		} catch (e) {
			sendFlash("error", `Failed to create new role: ${e}`);
		}
	}, [sendFlash, resetRoles]);

	const deleteMutation = useMutation(roleDelete, {
		onSuccess: () => {
			sendFlash("success", "Deleted role successfully");
			resetRoles();
		},
		onError: logErr,
	});

	const onEdit = useCallback(
		async (role: Role) => {
			try {
				await NiceModal.show(RoleEditModal, { role });
				sendFlash("success", `Role edited successfully: ${role.roleName}`);
				resetRoles();
			} catch (e) {
				sendFlash("error", `Failed to edit role: ${e}`);
			}
		},
		[sendFlash, resetRoles],
	);

	const onDelete = useCallback(
		async (role: Role) => {
			try {
				const confirm = Boolean(
					await NiceModal.show(ConfirmationModal, {
						title: "Are you sure you want to delete this role?",
					}),
				);
				if (!confirm) {
					return;
				}

				await deleteMutation.mutateAsync({ roleId: role.roleId });
			} catch (e) {
				sendFlash("error", `Error trying to delete role: ${e}`);
			}
		},
		[sendFlash, deleteMutation],
	);

	const table = useMaterialReactTable({
		...defaultOptions,
		columns,
		data: rolesResponse?.roles ?? [],
		rowCount: rolesResponse?.roles.length ?? 0,
		enableFilters: true,
		enableColumnFilters: true,
		state: {
			isLoading: isLoading,
			showAlertBanner: isError,
		},
		initialState: {
			showColumnFilters: true,
			columnVisibility: {
				roleId: true,
				roleName: true,
				permissions: true,
				userCount: true,
				createdOn: false,
				updatedOn: false,
			},
		},
		enableRowActions: true,
		displayColumnDefOptions: makeRowActionsDefOptions(2),
		renderRowActions: ({ row }) => (
			<RowActionContainer>
				<IconButton
					key="edit"
					color={"warning"}
					onClick={() => {
						onEdit(row.original);
					}}
				>
					<Tooltip title={`Edit Role`}>
						<EditIcon />
					</Tooltip>
				</IconButton>
				<IconButton
					key="logs"
					color={"error"}
					onClick={() => {
						onDelete(row.original);
					}}
				>
					<Tooltip title={"Delete Role"}>
						<RemoveIcon />
					</Tooltip>
				</IconButton>
			</RowActionContainer>
		),
	});

	return (
		<Grid container spacing={2}>
			<Grid size={{ xs: 12 }}>
				<RoleUsersTable />
			</Grid>
			<Grid size={{ xs: 12 }}>
				<SortableTable
					table={table}
					iconLeft={<GroupsIcon htmlColor={theme.palette.primary.contrastText} />}
					title={"Roles & Permissions"}
					buttons={[
						<Button key="add" color="success" startIcon={<AddIcon />} onClick={onCreate}>
							Create Role
						</Button>,
					]}
				/>
			</Grid>
			<Grid size={{ xs: 12 }}>
				<SMOverridesTable />
			</Grid>
			<Grid size={{ xs: 12 }}>
				<SMImmunityTable />
			</Grid>
		</Grid>
	);
}
