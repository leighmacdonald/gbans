import ChevronRightIcon from "@mui/icons-material/ChevronRight";
import ContentCopyIcon from "@mui/icons-material/ContentCopy";
import GroupsIcon from "@mui/icons-material/Groups";
import Button from "@mui/material/Button";
import IconButton from "@mui/material/IconButton";
import Link from "@mui/material/Link";
import Tooltip from "@mui/material/Tooltip";
import Typography from "@mui/material/Typography";
import { createMRTColumnHelper, type MRT_ColumnDef, useMaterialReactTable } from "material-react-table";
import { useCallback, useMemo } from "react";
import type { z } from "zod/v4";
import { cleanMapName } from "../api";
import { useAuth } from "../hooks/useAuth.ts";
import { useMapStateCtx } from "../hooks/useMapStateCtx.ts";
import { useQueueCtx } from "../hooks/useQueueCtx.ts";
import { useUserFlashCtx } from "../hooks/useUserFlashCtx.ts";
import { PermissionLevel } from "../schema/people.ts";
import type { schemaServerRow } from "../schema/server.ts";
import { tf2Fonts } from "../theme";
import { logErr } from "../util/errors";
import { Flag } from "./Flag";
import { StyledBadge } from "./StyledBadge.tsx";
import { createDefaultTableOptions } from "./table/options.ts";
import { SortableTable } from "./table/SortableTable.tsx";

type ServerRow = z.infer<typeof schemaServerRow>;

const columnHelper = createMRTColumnHelper<ServerRow>();
const defaultOptions = createDefaultTableOptions<ServerRow>();

export const ServerList = () => {
	const { sendFlash } = useUserFlashCtx();
	const { profile, hasPermission } = useAuth();
	const { selectedServers } = useMapStateCtx();
	const { joinQueue, leaveQueue, lobbies } = useQueueCtx();

	const metaServers = useMemo(() => {
		return selectedServers.map((s) => ({ ...s, copy: "", connect: "" }));
	}, [selectedServers]);

	const isQueued = useCallback(
		(server_id: number) => {
			try {
				return Boolean(
					lobbies
						.find((s) => s.server_id === server_id)
						?.members?.find((m) => m.steam_id === profile.steam_id),
				);
			} catch {
				return false;
			}
		},
		[lobbies, profile],
	);

	const columns = useMemo(
		() =>
			[
				columnHelper.accessor("cc", {
					header: "CC",
					size: 40,
					Cell: ({ cell }) => <Flag countryCode={cell.getValue()} />,
				}),
				columnHelper.accessor("name", {
					header: "Server",
					size: 450,
					Cell: ({ cell }) => (
						<Typography variant={"button"} fontFamily={tf2Fonts}>
							{cell.getValue()}
						</Typography>
					),
				}),
				columnHelper.accessor("map", {
					header: "Map",
					size: 150,
					Cell: ({ cell }) => <Typography variant={"body2"}>{cleanMapName(cell.getValue())}</Typography>,
				}),
				columnHelper.accessor("players", {
					header: "Players",
					size: 50,
					Cell: ({ row }) => (
						<Typography
							variant={"body2"}
						>{`${row.original.players + row.original.bots}/${row.original.max_players}`}</Typography>
					),
				}),
				columnHelper.accessor("distance", {
					header: "Dist",

					size: 60,
					meta: {
						tooltip: "Approximate distance from you",
					},
					Cell: ({ cell }) => (
						<Tooltip title={`Distance in hammer units: ${Math.round((cell.getValue() ?? 1) * 52.49)} khu`}>
							<Typography variant={"caption"}>{`${cell.getValue().toFixed(0)}km`}</Typography>
						</Tooltip>
					),
				}),
				columnHelper.display({
					header: "Cp",
					size: 30,
					meta: {
						tooltip: "Copy to clipboard",
					},
					Cell: ({ row }) => (
						<IconButton
							color={"primary"}
							aria-label={"Copy connect string to clipboard"}
							onClick={() => {
								navigator.clipboard
									.writeText(`connect ${row.original.ip}:${row.original.port}`)
									.then(() => {
										sendFlash("success", "Copied address to clipboard");
									})
									.catch((e) => {
										sendFlash("error", "Failed to copy address");
										logErr(e);
									});
							}}
						>
							<ContentCopyIcon />
						</IconButton>
					),
				}),
				hasPermission(PermissionLevel.Moderator)
					? columnHelper.display({
							header: "Queue",
							id: "queue",
							size: 30,
							Cell: ({ row }) => {
								const queued = isQueued(row.original.server_id);

								const count = lobbies
									? (lobbies.find((value) => {
											return value.server_id === row.original.server_id;
										})?.members?.length ?? 0)
									: 0;

								return (
									<Tooltip title="Join/Leave server queue. Number indicates actively queued players. (in testing)">
										<IconButton
											disabled={false}
											color={queued ? "success" : "primary"}
											onClick={() => {
												if (queued) {
													leaveQueue([String(row.original.server_id)]);
												} else {
													joinQueue([String(row.original.server_id)]);
												}
											}}
										>
											<StyledBadge badgeContent={count}>
												<GroupsIcon />
											</StyledBadge>
										</IconButton>
									</Tooltip>
								);
							},
						})
					: undefined,
				columnHelper.accessor("connect", {
					header: "Connect",
					size: 125,
					Cell: ({ row }) => (
						<Button
							fullWidth
							endIcon={<ChevronRightIcon />}
							component={Link}
							href={`steam://run/440//+connect ${row.original.ip}:${row.original.port}`}
							variant={"contained"}
							sx={{ minWidth: 100 }}
						>
							Join
						</Button>
					),
				}),
			].filter((f) => f) as Array<MRT_ColumnDef<ServerRow>>,
		[lobbies, hasPermission, joinQueue, leaveQueue, sendFlash, isQueued],
	);

	const table = useMaterialReactTable({
		...defaultOptions,
		columns: columns,
		data: metaServers ?? [],
		enableFilters: false,
		enableColumnFilters: false,
		enableSorting: false,
		enableRowActions: false,
		enableColumnActions: false,
		enablePagination: false,
		initialState: {
			...defaultOptions.initialState,
			sorting: [{ id: "distance", desc: true }],
			columnVisibility: {
				name: true,
			},
		},
	});

	return <SortableTable table={table} title={"Servers"} hideToolbarButtons />;
};
