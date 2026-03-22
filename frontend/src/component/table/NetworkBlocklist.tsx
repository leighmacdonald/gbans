import NiceModal, { useModal } from "@ebay/nice-modal-react";
import AddIcon from "@mui/icons-material/Add";
import DeleteIcon from "@mui/icons-material/Delete";
import EditIcon from "@mui/icons-material/Edit";
import { Link } from "@mui/material";
import Grid from "@mui/material/Grid";
import IconButton from "@mui/material/IconButton";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { createMRTColumnHelper, useMaterialReactTable } from "material-react-table";
import { useCallback, useMemo } from "react";
import { apiDeleteCIDRBlockSource, apiGetCIDRBlockLists } from "../../api/network";
import { useUserFlashCtx } from "../../hooks/useUserFlashCtx";
import type { CIDRBlockSource } from "../../schema/network";
import { logErr } from "../../util/errors";
import { renderDate } from "../../util/time";
import { CIDRBlockEditorModal } from "../modal/CIDRBlockEditorModal";
import { ConfirmationModal } from "../modal/ConfirmationModal";
import { BoolCell } from "./BoolCell";
import { createDefaultTableOptions } from "./options";
import { SortableTable } from "./SortableTable";

const columnHelper = createMRTColumnHelper<CIDRBlockSource>();
const defaultOptions = createDefaultTableOptions<CIDRBlockSource>();

export const NetworkBlocklist = () => {
	const { sendFlash, sendError } = useUserFlashCtx();
	const confirmModal = useModal(ConfirmationModal);
	const queryClient = useQueryClient();

	const { data, isLoading, isError } = useQuery({
		queryKey: ["networkBlockListSources"],
		queryFn: async () => {
			return await apiGetCIDRBlockLists();
		},
	});

	const sourceMutation = useMutation({
		mutationKey: ["networkBlockSourceDelete"],
		mutationFn: async (variables: { cidr_block_source_id: number }) => {
			await apiDeleteCIDRBlockSource(variables.cidr_block_source_id);
		},
		onSuccess: (_, variables) => {
			sendFlash("success", "Blocklist source deleted");
			queryClient.setQueryData(
				["networkBlockListSources"],
				data?.filter((b) => b.cidr_block_source_id !== variables.cidr_block_source_id),
			);
		},
		onError: sendError,
	});

	const onDelete = useCallback(
		async (cidr_block_source_id: number) => {
			try {
				const confirmed = await confirmModal.show({
					title: "Delete CIDR Block Source?",
					children: "This action is permanent",
				});
				if (confirmed) {
					sourceMutation.mutate({ cidr_block_source_id });
				}
				await confirmModal.hide();
			} catch (e) {
				logErr(e);
			}
		},
		[confirmModal, sourceMutation],
	);

	const onEdit = useCallback(
		async (source?: CIDRBlockSource) => {
			try {
				const updated = (await NiceModal.show(CIDRBlockEditorModal, {
					source,
				})) as CIDRBlockSource;

				queryClient.setQueryData(
					["networkBlockListSources"],
					(data ?? []).map((bs) => {
						return bs.cidr_block_source_id === updated.cidr_block_source_id ? updated : bs;
					}),
				);
			} catch (e) {
				logErr(e);
			}
		},
		[data, queryClient],
	);

	const columns = useMemo(
		() => [
			columnHelper.accessor("name", {
				header: "Name",
				grow: false,
			}),
			columnHelper.accessor("url", {
				header: "URL",
				grow: true,
				Cell: ({ cell, renderedCellValue }) => <Link href={cell.getValue()}>{renderedCellValue}</Link>,
			}),
			columnHelper.accessor("enabled", {
				header: "Enabled",
				grow: false,
				filterVariant: "checkbox",
				Cell: ({ cell }) => <BoolCell enabled={cell.getValue()} />,
			}),
			columnHelper.accessor("created_on", {
				header: "Updated",
				grow: false,
				Cell: ({ cell }) => renderDate(cell.getValue()),
			}),
		],
		[],
	);

	const table = useMaterialReactTable({
		...defaultOptions,
		columns,
		data: data ?? [],
		enableFilters: true,
		enableHiding: true,
		enableFacetedValues: true,
		state: {
			isLoading,
			showAlertBanner: isError,
		},
		initialState: {
			...defaultOptions.initialState,
			sorting: [{ id: "name", desc: false }],
			columnVisibility: {
				name: true,
				url: true,
				enabled: true,
				created_on: true,
			},
		},
		enableRowActions: true,
		renderRowActionMenuItems: ({ row }) => [
			<IconButton
				key={"delete"}
				color={"error"}
				onClick={async () => {
					await onDelete(row.original.cidr_block_source_id);
				}}
			>
				<DeleteIcon />
			</IconButton>,
			<IconButton
				key={"edit"}
				color={"warning"}
				onClick={async () => {
					await onEdit(row.original);
				}}
			>
				<EditIcon />
			</IconButton>,
		],
	});

	return (
		<Grid container spacing={2}>
			<Grid size={{ xs: 12 }}>
				<SortableTable
					table={table}
					title={"CIDR Block Sources"}
					buttons={[
						<IconButton
							key={"add"}
							onClick={async () => {
								await onEdit();
							}}
							sx={{ color: "primary.contrastText" }}
						>
							<AddIcon />
						</IconButton>,
					]}
				/>
			</Grid>
		</Grid>
	);
};
