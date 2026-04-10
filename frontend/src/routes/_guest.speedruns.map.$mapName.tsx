import Typography from "@mui/material/Typography";
import { useQuery } from "@tanstack/react-query";
import { createFileRoute } from "@tanstack/react-router";
import { createMRTColumnHelper, useMaterialReactTable } from "material-react-table";
import { useMemo } from "react";
import { apiGetServers, getSpeedrunsTopMap } from "../api";
import { TextLink } from "../component/TextLink.tsx";
import { createDefaultTableOptions } from "../component/table/options.ts";
import { SortableTable } from "../component/table/SortableTable.tsx";
import { TableCellSmall } from "../component/table/TableCellSmall.tsx";
import { TableCellString } from "../component/table/TableCellString.tsx";
import type { SpeedrunMapOverview } from "../schema/speedrun.ts";
import { ensureFeatureEnabled } from "../util/features.ts";
import { durationString, renderDateTime } from "../util/time.ts";

const columnHelper = createMRTColumnHelper<SpeedrunMapOverview>();
const defaultOptions = createDefaultTableOptions<SpeedrunMapOverview>();

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

	const { data, isLoading, isError } = useQuery({
		queryKey: ["speedruns_map", mapName],
		queryFn: ({ signal }) => {
			return getSpeedrunsTopMap(mapName, signal);
		},
	});

	const {
		data: servers,
		isLoading: isLoadingServers,
		isError: isErrorServers,
	} = useQuery({
		queryKey: ["serversSimple"],
		queryFn: ({ signal }) => apiGetServers(signal),
	});

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
							params={{ speedrunId: String(row.original.speedrun_id) }}
						>
							{value}
						</TextLink>
					);
				},
			}),
			columnHelper.accessor("initial_rank", {
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
			columnHelper.accessor("player_count", {
				header: "Max Players",
				size: 100,
				Cell: ({ cell }) => {
					return <TableCellString>{cell.getValue()}</TableCellString>;
				},
			}),
			columnHelper.accessor("bot_count", {
				header: "Max Bots",
				size: 100,
				Cell: ({ cell }) => {
					return <TableCellString>{cell.getValue()}</TableCellString>;
				},
			}),
			columnHelper.accessor("total_players", {
				header: "Total Players",
				size: 100,
				Cell: ({ cell }) => {
					return <TableCellString>{cell.getValue()}</TableCellString>;
				},
			}),
			columnHelper.accessor("server_id", {
				header: "Server",
				size: 100,
				Cell: ({ cell }) => {
					const srv = (servers ?? []).find((s) => (s.server_id = cell.getValue()));
					return <TableCellString>{srv?.server_name}</TableCellString>;
				},
			}),
			columnHelper.accessor("created_on", {
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
		data: data ?? [],
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
