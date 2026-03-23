import NiceModal from "@ebay/nice-modal-react";
import AssuredWorkloadIcon from "@mui/icons-material/AssuredWorkload";
import DeleteIcon from "@mui/icons-material/Delete";
import IconButton from "@mui/material/IconButton";
import Tooltip from "@mui/material/Tooltip";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { createMRTColumnHelper, useMaterialReactTable } from "material-react-table";
import { useCallback, useMemo } from "react";
import { apiDeleteSMGroupImmunity, apiGetSMGroupImmunities, apiGetSMGroups } from "../../api";
import { useUserFlashCtx } from "../../hooks/useUserFlashCtx";
import type { SMGroupImmunity } from "../../schema/sourcemod.ts";
import { logErr } from "../../util/errors";
import { renderDateTime } from "../../util/time.ts";
import { ConfirmationModal } from "../modal/ConfirmationModal.tsx";
import { SMGroupImmunityCreateModal } from "../modal/SMGroupImmunityCreateModal.tsx";
import { createDefaultTableOptions } from "./options.ts";
import { SortableTable } from "./SortableTable.tsx";
import { TableCellString } from "./TableCellString";

const columnHelper = createMRTColumnHelper<SMGroupImmunity>();
const defaultOptions = createDefaultTableOptions<SMGroupImmunity>();

export const SMImmunityTable = () => {
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
		data: immunities,
		isLoading: isLoadingImmunities,
		isError: isErrorImmunities,
	} = useQuery({
		queryKey: ["serverImmunities"],
		queryFn: async ({ signal }) => {
			return await apiGetSMGroupImmunities(signal);
		},
	});

	const onCreateImmunity = useCallback(async () => {
		try {
			const immunity = (await NiceModal.show(SMGroupImmunityCreateModal, { groups })) as SMGroupImmunity;
			queryClient.setQueryData(["serverImmunities"], [...(immunities ?? []), immunity]);
			sendFlash("success", `Group immunity created successfully: ${immunity.group_immunity_id}`);
		} catch (e) {
			logErr(e);
			sendFlash("error", "Error trying to add group immunity");
		}
	}, [groups, immunities, queryClient, sendFlash]);

	const delImmunityMutation = useMutation({
		mutationKey: ["delGroupImmunity"],
		mutationFn: async ({ immunity }: { immunity: SMGroupImmunity }) => {
			const ac = new AbortController();
			await apiDeleteSMGroupImmunity(immunity.group_immunity_id, ac.signal);
			return immunity;
		},
		onSuccess: (deleted) => {
			queryClient.setQueryData(
				["serverImmunities"],
				(immunities ?? []).filter((o) => {
					return o.group_immunity_id !== deleted.group_immunity_id;
				}),
			);
			sendFlash("success", `Group immunity deleted successfully: ${deleted.group_immunity_id}`);
		},
		onError: sendError,
	});

	const onDelete = useCallback(
		async (immunity: SMGroupImmunity) => {
			try {
				const confirmed = (await NiceModal.show(ConfirmationModal, {
					title: "Delete group immunity?",
					children: "This cannot be undone",
				})) as boolean;
				if (!confirmed) {
					return;
				}
				delImmunityMutation.mutate({ immunity });
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
			columnHelper.accessor("created_on", {
				header: "Created On",
				grow: false,
				Cell: ({ cell }) => <TableCellString>{renderDateTime(cell.getValue())}</TableCellString>,
			}),
		],
		[],
	);

	const table = useMaterialReactTable({
		...defaultOptions,
		columns,
		data: immunities ?? [],
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
					disabled={(groups ?? []).length < 2}
					key="create"
					sx={{ color: "primary.contrastText" }}
				>
					<AssuredWorkloadIcon />
				</IconButton>,
			]}
		/>
	);
};
