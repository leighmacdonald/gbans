import { useMutation, useQuery } from "@connectrpc/connect-query";
import NiceModal from "@ebay/nice-modal-react";
import AssuredWorkloadIcon from "@mui/icons-material/AssuredWorkload";
import DeleteIcon from "@mui/icons-material/Delete";
import IconButton from "@mui/material/IconButton";
import Tooltip from "@mui/material/Tooltip";
import { useQueryClient } from "@tanstack/react-query";
import { createMRTColumnHelper, useMaterialReactTable } from "material-react-table";
import { useCallback, useMemo } from "react";
import { useUserFlashCtx } from "../../hooks/useUserFlashCtx";
import type { GroupImmunity } from "../../rpc/sourcemod/v1/sourcemod_pb.ts";
import {
	deleteImmunity,
	groupImmunities,
	groups,
} from "../../rpc/sourcemod/v1/sourcemod-SourcemodService_connectquery.ts";
import { logErr } from "../../util/errors";
import { renderTimestamp } from "../../util/time.ts";
import { ConfirmationModal } from "../modal/ConfirmationModal.tsx";
import { SMGroupImmunityCreateModal } from "../modal/SMGroupImmunityCreateModal.tsx";
import { createDefaultTableOptions } from "./options.ts";
import { SortableTable } from "./SortableTable.tsx";
import { TableCellString } from "./TableCellString";

const columnHelper = createMRTColumnHelper<GroupImmunity>();
const defaultOptions = createDefaultTableOptions<GroupImmunity>();

export const SMImmunityTable = () => {
	const { sendFlash, sendError } = useUserFlashCtx();
	const queryClient = useQueryClient();

	const { data: groupList, isLoading: isLoadingGroups, isError: isErrorGroups } = useQuery(groups);

	const { data: immunities, isLoading: isLoadingImmunities, isError: isErrorImmunities } = useQuery(groupImmunities);

	const onCreateImmunity = useCallback(async () => {
		try {
			const immunity = (await NiceModal.show(SMGroupImmunityCreateModal, {
				groups: groupList?.groups,
			})) as GroupImmunity;
			queryClient.setQueryData(["serverImmunities"], [...(immunities?.groupImmunities ?? []), immunity]);
			sendFlash("success", `Group immunity created successfully: ${immunity.groupImmunityId}`);
		} catch (e) {
			logErr(e);
			sendFlash("error", "Error trying to add group immunity");
		}
	}, [immunities, queryClient, sendFlash, groupList?.groups]);

	// FIXME should this be a separate group immunity?
	const delImmunityMutation = useMutation(deleteImmunity, {
		onSuccess: (_, deleted) => {
			queryClient.setQueryData(
				["serverImmunities"],
				(immunities?.groupImmunities ?? []).filter((o) => {
					return o.groupImmunityId !== deleted.immunityId;
				}),
			);
			sendFlash("success", `Group immunity deleted successfully: ${deleted.immunityId}`);
		},
		onError: sendError,
	});

	const onDelete = useCallback(
		async (immunity: GroupImmunity) => {
			try {
				const confirmed = (await NiceModal.show(ConfirmationModal, {
					title: "Delete group immunity?",
					children: "This cannot be undone",
				})) as boolean;
				if (!confirmed) {
					return;
				}
				delImmunityMutation.mutate({ immunityId: immunity.groupImmunityId });
			} catch (e) {
				sendFlash("error", `Failed to create confirmation modal: ${e}`);
			}
		},
		[delImmunityMutation, sendFlash],
	);

	const columns = useMemo(
		() => [
			columnHelper.accessor("group.name", {
				header: "Group",
				grow: true,
				Cell: ({ cell }) => <TableCellString>{cell.getValue()}</TableCellString>,
			}),
			columnHelper.accessor("other.name", {
				header: "Immunity From",
				grow: true,
				Cell: ({ cell }) => <TableCellString>{cell.getValue()}</TableCellString>,
			}),
			columnHelper.accessor("createdOn", {
				header: "Created On",
				grow: false,
				Cell: ({ cell }) => <TableCellString>{renderTimestamp(cell.getValue())}</TableCellString>,
			}),
		],
		[],
	);

	const table = useMaterialReactTable({
		...defaultOptions,
		columns,
		data: immunities?.groupImmunities ?? [],
		enableFilters: true,
		enableRowActions: true,
		state: {
			isLoading: isLoadingImmunities || isLoadingGroups,
			showAlertBanner: isErrorImmunities || isErrorGroups,
		},
		renderRowActionMenuItems: ({ row }) => [
			<Tooltip title={"Delete override"} key={"delete"}>
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
			title={"Group Immunities"}
			buttons={[
				<IconButton
					onClick={onCreateImmunity}
					disabled={(groupList?.groups ?? []).length < 2}
					key="create"
					sx={{ color: "primary.contrastText" }}
				>
					<AssuredWorkloadIcon />
				</IconButton>,
			]}
		/>
	);
};
