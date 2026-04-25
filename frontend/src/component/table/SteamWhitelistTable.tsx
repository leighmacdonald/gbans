import { useMutation, useQuery } from "@connectrpc/connect-query";
import NiceModal, { useModal } from "@ebay/nice-modal-react";
import AddIcon from "@mui/icons-material/Add";
import DeleteIcon from "@mui/icons-material/Delete";
import IconButton from "@mui/material/IconButton";
import TableCell from "@mui/material/TableCell";
import Typography from "@mui/material/Typography";
import { Grid } from "@mui/system";
import { useQueryClient } from "@tanstack/react-query";
import { createMRTColumnHelper, useMaterialReactTable } from "material-react-table";
import { useCallback, useMemo } from "react";
import { useUserFlashCtx } from "../../hooks/useUserFlashCtx";
import type { WhitelistSteam } from "../../rpc/network/v1/blocklist_pb.ts";
import { whitelistSteam, whitelistSteamDelete } from "../../rpc/network/v1/blocklist-BlocklistService_connectquery.ts";
import { logErr } from "../../util/errors";
import { renderTimestamp } from "../../util/time";
import { ConfirmationModal } from "../modal/ConfirmationModal";
import { SteamWhitelistEditorModal } from "../modal/SteamWhitelistEditorModal";
import { PersonCell } from "../PersonCell";
import { createDefaultTableOptions } from "./options";
import { SortableTable } from "./SortableTable";

const columnHelperSteam = createMRTColumnHelper<WhitelistSteam>();
const defaultOptionsSteam = createDefaultTableOptions<WhitelistSteam>();

export const SteamWhitelistTable = () => {
	const confirmModal = useModal(ConfirmationModal);
	const queryClient = useQueryClient();
	const { sendFlash, sendError } = useUserFlashCtx();

	const { data, isLoading, isError } = useQuery(whitelistSteam);

	const steamWhitelistDelete = useMutation(whitelistSteamDelete, {
		onSuccess: () => {
			sendFlash("success", "Steam whitelist deleted");
		},
		onError: sendError,
	});

	const onEdit = useCallback(async () => {
		try {
			const newSource = (await NiceModal.show(SteamWhitelistEditorModal, {})) as WhitelistSteam;

			queryClient.setQueryData(
				["networkSteamWhitelist"],
				(data?.whitelists ?? []).map((src) => {
					return src.steamId === newSource.steamId ? newSource : src;
				}),
			);
			sendFlash("success", "Steam whitelist added");
		} catch (e) {
			sendFlash("error", `Failed to add steam whitelist: ${e}`);
		}
	}, [queryClient, sendFlash, data]);

	const onDelete = useCallback(
		async (wl: WhitelistSteam) => {
			try {
				const confirmed = await confirmModal.show({
					title: "Delete steam whitelist?",
					children: "This action is permanent",
				});
				if (confirmed) {
					steamWhitelistDelete.mutate({ steamId: wl.steamId });
				}
				await confirmModal.hide();
			} catch (e) {
				logErr(e);
			}
		},
		[confirmModal, steamWhitelistDelete],
	);

	const columns = useMemo(
		() => [
			columnHelperSteam.accessor("steamId", {
				header: "Steam ID",
				grow: true,
				Cell: ({ row }) => (
					<PersonCell
						steam_id={row.original.steamId}
						avatar_hash={row.original.avatarHash}
						personaname={row.original.personaName}
					/>
				),
			}),
			columnHelperSteam.accessor("createdOn", {
				header: "Updated",
				grow: false,
				Cell: ({ cell }) => (
					<TableCell>
						<Typography>{renderTimestamp(cell.getValue())}</Typography>
					</TableCell>
				),
			}),
		],
		[],
	);

	const table = useMaterialReactTable({
		...defaultOptionsSteam,
		columns,
		data: data?.whitelists ?? [],
		enableFilters: true,
		enableHiding: true,
		enableFacetedValues: true,
		state: {
			isLoading,
			showAlertBanner: isError,
		},
		initialState: {
			...defaultOptionsSteam.initialState,
			sorting: [{ id: "ban_id", desc: true }],
			columnVisibility: {
				source_id: false,
				target_id: true,
				reason: true,
			},
		},
		enableRowActions: true,
		renderRowActionMenuItems: ({ row }) => [
			<IconButton
				key={"delete"}
				color={"error"}
				onClick={async () => {
					await onDelete(row.original);
				}}
			>
				<DeleteIcon />
			</IconButton>,
		],
	});

	return (
		<Grid container spacing={2}>
			<Grid size={{ xs: 12 }}>
				<SortableTable
					table={table}
					title={"Steam Whitelist"}
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
