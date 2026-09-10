import { useQuery } from "@connectrpc/connect-query";
import NiceModal from "@ebay/nice-modal-react";
import AddIcon from "@mui/icons-material/Add";
import EditIcon from "@mui/icons-material/Edit";
import PeopleIcon from "@mui/icons-material/People";
import { useTheme } from "@mui/material";
import Button from "@mui/material/Button";
import IconButton from "@mui/material/IconButton";
import Tooltip from "@mui/material/Tooltip";
import { createMRTColumnHelper, useMaterialReactTable } from "material-react-table";
import { useCallback, useMemo } from "react";
import { useUserFlashCtx } from "../../hooks/useUserFlashCtx.ts";
import type { Role, RoleUser } from "../../rpc/roles/v1/roles_pb.ts";
import { roleUsers } from "../../rpc/roles/v1/roles-RolesService_connectquery.ts";
import { AssignRolesModal } from "../modal/AssignRolesModal.tsx";
import { RowActionContainer } from "../RowActionContainer.tsx";
import { createDefaultTableOptions, makeRowActionsDefOptions } from "./options.ts";
import { SortableTable } from "./SortableTable.tsx";
import { TableCellString } from "./TableCellString.tsx";

const columnHelper = createMRTColumnHelper<RoleUser>();
const defaultOptions = createDefaultTableOptions<RoleUser>();

export const RoleUsersTable = () => {
	const { sendFlash } = useUserFlashCtx();
	const theme = useTheme();

	const { data: usersResponse, isLoading, isError } = useQuery(roleUsers, {});
	const users = usersResponse?.users ?? [];

	const onCreate = useCallback(async () => {
		try {
			await NiceModal.show(AssignRolesModal, {});
		} catch (e) {
			sendFlash("error", `Failed to assign roles: ${e}`);
		}
	}, [sendFlash]);

	const onEdit = useCallback(
		async (user: RoleUser) => {
			try {
				await NiceModal.show(AssignRolesModal, { steamId: user.steamId, roles: user.roles });
			} catch (e) {
				sendFlash("error", `Failed to edit roles: ${e}`);
			}
		},
		[sendFlash],
	);

	const roleNameOptions = useMemo(() => {
		const names = new Set<string>();
		for (const user of users) {
			for (const role of user.roles) {
				names.add(role.roleName);
			}
		}

		return [...names].map((name) => ({ value: name, label: name }));
	}, [users]);

	const columns = useMemo(
		() => [
			columnHelper.accessor("steamId", {
				header: "Steam ID",
				grow: false,
				size: 180,
				enableColumnFilter: true,
				Cell: ({ cell }) => <TableCellString>{cell.getValue()}</TableCellString>,
			}),
			columnHelper.accessor("personaName", {
				header: "Persona Name",
				grow: true,
				enableColumnFilter: true,
				Cell: ({ cell }) => <TableCellString>{cell.getValue()}</TableCellString>,
			}),
			columnHelper.accessor("roles", {
				header: "Roles",
				grow: true,
				enableSorting: false,
				filterVariant: "multi-select",
				filterSelectOptions: roleNameOptions,
				filterFn: (row, _column, value) => {
					if (!value || value.length === 0) {
						return true;
					}

					const names = (row.getValue("roles") as Role[]).map((r) => r.roleName);
					return (value as string[]).some((n) => names.includes(n));
				},
				Cell: ({ cell }) => (
					<TableCellString>{(cell.getValue() as Role[]).map((r) => r.roleName).join(", ")}</TableCellString>
				),
			}),
		],
		[roleNameOptions],
	);

	const table = useMaterialReactTable({
		...defaultOptions,
		columns,
		data: users,
		rowCount: users.length,
		enableFilters: true,
		enableColumnFilters: true,
		state: {
			isLoading,
			showAlertBanner: isError,
		},
		enableRowActions: true,
		displayColumnDefOptions: makeRowActionsDefOptions(1),
		renderRowActions: ({ row }) => (
			<RowActionContainer>
				<IconButton
					key="edit"
					color={"warning"}
					onClick={() => {
						onEdit(row.original);
					}}
				>
					<Tooltip title={"Edit Roles"}>
						<EditIcon />
					</Tooltip>
				</IconButton>
			</RowActionContainer>
		),
		initialState: {
			showColumnFilters: true,
			columnVisibility: {
				steamId: true,
				personaName: true,
				roles: true,
			},
		},
	});

	return (
		<SortableTable
			table={table}
			iconLeft={<PeopleIcon htmlColor={theme.palette.primary.contrastText} />}
			title={"Persons with Roles"}
			buttons={[
				<Button key="assign" color="success" startIcon={<AddIcon />} onClick={onCreate}>
					Assign Roles
				</Button>,
			]}
		/>
	);
};
