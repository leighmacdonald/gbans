import Typography from "@mui/material/Typography";
import { createFileRoute } from "@tanstack/react-router";
import { createMRTColumnHelper, useMaterialReactTable } from "material-react-table";
import { useMemo } from "react";
import { TextLink } from "../component/TextLink.tsx";
import { createDefaultTableOptions } from "../component/table/options.ts";
import { SortableTable } from "../component/table/SortableTable.tsx";
import { TableCellSmall } from "../component/table/TableCellSmall.tsx";
import { TableCellString } from "../component/table/TableCellString.tsx";
import { ensureFeatureEnabled } from "../util/features.ts";
import { durationString, renderDateTime } from "../util/time.ts";
import { mapSpeedruns } from "../rpc/servers/v1/speedruns-SpeedrunsService_connectquery.ts";
import type { SpeedrunOverview } from "../rpc/servers/v1/speedruns_pb.ts";
import { useQuery } from "@connectrpc/connect-query";
import { servers } from "../rpc/servers/v1/servers-ServersService_connectquery.ts";

const columnHelper = createMRTColumnHelper<SpeedrunOverview>();
const defaultOptions = createDefaultTableOptions<SpeedrunOverview>();

export const Route = createFileRoute("/_guest/speedruns/map/$mapName")({
	component: SpeedrunsMap,
	beforeLoad: ({ context }) => {
		ensureFeatureEnabled(context.appInfo.speedrunsEnabled);
	},
	head: ({ match }) => ({
		meta: [{ name: "description", content: "Map Speedruns" }, match.context.title("Map Speedruns")],
	}),
});

function SpeedrunsMap() {
	const { mapName } = Route.useParams();

	const { data, isLoading, isError } = useQuery(mapSpeedruns, { mapName });
	const { data: serverList, isLoading: isLoadingServers, isError: isErrorServers } = useQuery(servers);

	const columns = useMemo(
		() => [
			columnHelper.accessor("rank", {
				header: "Rank",
				size: 5,
				Cell: ({ cell, row }) => {
					const value = cell.getValue();
					const weight = value <= 3 ? 700 : 500;
					return (
						<TextLink
							fontWeight={weight}
							textAlign={"right"}
							paddingRight={2}
							to={"/speedruns/id/$speedrunId"}
							params={{ speedrunId: String(row.original.speedrunId) }}
						>
							{value}
						</TextLink>
					);
				},
			}),
			columnHelper.accessor("initialRank", {
				header: "High",
				size: 10,
				Cell: ({ cell }) => (
					<TableCellSmall>
						<Typography align={"center"}>{cell.getValue()}</Typography>
					</TableCellSmall>
				),
			}),
			columnHelper.accessor("duration", {
				header: "Time",
				size: 100,
				Cell: ({ cell }) => (
					<TableCellSmall>
						<Typography align={"center"}>{durationString(cell.getValue() / 1000)}</Typography>
					</TableCellSmall>
				),
			}),
			columnHelper.accessor("playerCount", {
				header: "Max Players",
				size: 100,
				Cell: ({ cell }) => {
					return <TableCellString>{cell.getValue()}</TableCellString>;
				},
			}),
			columnHelper.accessor("botCount", {
				header: "Max Bots",
				size: 100,
				Cell: ({ cell }) => {
					return <TableCellString>{cell.getValue()}</TableCellString>;
				},
			}),
			columnHelper.accessor("totalPlayers", {
				header: "Total Players",
				size: 100,
				Cell: ({ cell }) => {
					return <TableCellString>{cell.getValue()}</TableCellString>;
				},
			}),
			columnHelper.accessor("serverId", {
				header: "Server",
				size: 100,
				Cell: ({ cell }) => {
					const srv = (serverList?.servers ?? []).find((s) => (s.serverId = cell.getValue()));
					return <TableCellString>{srv?.serverName}</TableCellString>;
				},
			}),
			columnHelper.accessor("createdOn", {
				header: "Submitted",
				size: 100,
				Cell: ({ cell }) => {
					return <TableCellString>{renderDateTime(cell.getValue())}</TableCellString>;
				},
			}),
		],
		[servers],
	);

	const table = useMaterialReactTable({
		...defaultOptions,
		columns,
		data: data?.speedruns ?? [],
		enableFilters: true,
		enableRowActions: true,
		state: {
			isLoading: isLoading || isLoadingServers,
			showAlertBanner: isError || isErrorServers,
		},

		initialState: {
			...defaultOptions.initialState,
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

	return <SortableTable table={table} title={`Speedruns: ${mapName}`} />;
}
