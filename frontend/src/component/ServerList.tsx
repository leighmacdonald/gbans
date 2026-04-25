import ChevronRightIcon from "@mui/icons-material/ChevronRight";
import ContentCopyIcon from "@mui/icons-material/ContentCopy";
import Button from "@mui/material/Button";
import IconButton from "@mui/material/IconButton";
import Link from "@mui/material/Link";
import Tooltip from "@mui/material/Tooltip";
import Typography from "@mui/material/Typography";
import { createMRTColumnHelper, type MRT_ColumnDef, useMaterialReactTable } from "material-react-table";
import { useMemo } from "react";
import { useMapStateCtx } from "../hooks/useMapStateCtx.ts";
import { useUserFlashCtx } from "../hooks/useUserFlashCtx.ts";
import type { SafeServer } from "../rpc/servers/v1/servers_pb.ts";
import { tf2Fonts } from "../theme";
import { logErr } from "../util/errors";
import { cleanMapName } from "../util/strings.ts";
import { Flag } from "./Flag";
import { createDefaultTableOptions } from "./table/options.ts";
import { SortableTable } from "./table/SortableTable.tsx";

const columnHelper = createMRTColumnHelper<SafeServer>();
const defaultOptions = createDefaultTableOptions<SafeServer>();

export const ServerList = () => {
	const { sendFlash } = useUserFlashCtx();
	const { selectedServers } = useMapStateCtx();

	const metaServers = useMemo(() => {
		return selectedServers.map((s) => ({ ...s, copy: "", connect: "" }));
	}, [selectedServers]);

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
						>{`${row.original.players}/${Number(row.original.maxPlayers) > 0 ? row.original.maxPlayers : row.original.maxPlayers}`}</Typography>
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
				columnHelper.display({
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
			].filter((f) => f) as Array<MRT_ColumnDef<SafeServer>>,
		[sendFlash],
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
			pagination: {
				pageSize: 100,
				pageIndex: 0,
			},
			sorting: [{ id: "distance", desc: true }],
			columnVisibility: {
				name: true,
			},
		},
	});

	return <SortableTable table={table} title={"Servers"} hideToolbarButtons />;
};
