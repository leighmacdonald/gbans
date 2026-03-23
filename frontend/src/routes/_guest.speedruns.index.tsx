import ElectricBoltIcon from "@mui/icons-material/ElectricBolt";
import Grid from "@mui/material/Grid";
import TableCell from "@mui/material/TableCell";
import Typography from "@mui/material/Typography";
import { useQuery } from "@tanstack/react-query";
import { createFileRoute } from "@tanstack/react-router";
import { createMRTColumnHelper, useMaterialReactTable } from "material-react-table";
import { useMemo } from "react";
import { apiGetServers, getSpeedrunsRecent } from "../api";
import { ContainerWithHeader } from "../component/ContainerWithHeader.tsx";
import { TextLink } from "../component/TextLink.tsx";
import { createDefaultTableOptions } from "../component/table/options.ts";
import { SortableTable } from "../component/table/SortableTable.tsx";
import { TableCellSmall } from "../component/table/TableCellSmall.tsx";
import { TableCellString } from "../component/table/TableCellString.tsx";
import type { SpeedrunMapOverview } from "../schema/speedrun.ts";
import { durationString, renderDateTime } from "../util/time.ts";

export const Route = createFileRoute("/_guest/speedruns/")({
	component: SpeedrunsIndex,
	head: ({ match }) => ({
		meta: [{ name: "description", content: "Speedruns Overall Results" }, match.context.title("Speedruns")],
	}),
});

function SpeedrunsIndex() {
	return (
		<Grid container spacing={2}>
			<Grid size={{ xs: 12, md: 4 }}>
				<ContainerWithHeader title={"Speedruns"} iconLeft={<ElectricBoltIcon />}>
					<Typography>
						These are the overall results for the speedruns. Speedruns are automatically created upon match
						completion. For a player to count in the overall participants, they must have played a minimum
						of 25% of the total play time of the map.
					</Typography>
				</ContainerWithHeader>
			</Grid>

			<Grid size={{ xs: 12, md: 8 }}>
				<SpeedrunRecentTable />
			</Grid>

			{/*{speedruns &&
				keys.map((map_name) => {
					return (
						<Grid size={{ xs: 12, md: 6, lg: 4 }} key={`map-${map_name}`}>
							<ContainerWithHeaderAndButtons
								title={map_name}
								iconLeft={<EmojiEventsIcon />}
								buttons={[
									<ButtonGroup key={"buttons"}>
										<ButtonLink
											variant={"contained"}
											color={"success"}
											endIcon={<PageviewIcon />}
											to={"/speedruns/map/$mapName"}
											params={{ mapName: map_name }}
										>
											More
										</ButtonLink>
									</ButtonGroup>,
								]}
							>
								<SpeedrunTopTable
									speedruns={speedruns[map_name]}
									servers={servers}
									isLoading={isLoading || isLoadingServers}
								></SpeedrunTopTable>
							</ContainerWithHeaderAndButtons>
						</Grid>
					);
				})}*/}
		</Grid>
	);
}

const columnHelper = createMRTColumnHelper<SpeedrunMapOverview>();
const defaultOptions = createDefaultTableOptions<SpeedrunMapOverview>();

const SpeedrunRecentTable = () => {
	const recentCount = 10;
	const {
		data: recent,
		isLoading: isLoadingRecent,
		isError: isErrorRecent,
	} = useQuery({
		queryKey: ["speedruns_recent", recentCount],
		queryFn: async ({ signal }) => {
			return await getSpeedrunsRecent(recentCount, signal);
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
				size: 10,
				Cell: ({ cell }) => {
					const value = cell.getValue();
					const weight = value <= 3 ? 700 : 500;
					return (
						<TableCell>
							<Typography fontWeight={weight}>{value}</Typography>
						</TableCell>
					);
				},
			}),
			columnHelper.accessor("speedrun_id", {
				header: "ID",
				size: 10,
				Cell: ({ cell, row }) => {
					return (
						<TableCell>
							<TextLink
								fontWeight={700}
								to={"/speedruns/id/$speedrunId"}
								params={{ speedrunId: String(row.original.speedrun_id) }}
							>
								{cell.getValue()}
							</TextLink>
						</TableCell>
					);
				},
			}),
			columnHelper.accessor("map_detail", {
				header: "Map",
				size: 60,
				Cell: ({ cell }) => (
					<TableCellSmall>
						<Typography align={"center"}>{cell.getValue().map_name}</Typography>
					</TableCellSmall>
				),
			}),
			columnHelper.accessor("duration", {
				header: "Time",
				size: 60,
				Cell: ({ cell }) => (
					<TableCellSmall>
						<Typography align={"center"}>{durationString(cell.getValue())}</Typography>
					</TableCellSmall>
				),
			}),
			columnHelper.accessor("total_players", {
				header: "Players",
				size: 30,
				Cell: ({ cell }) => {
					return <TableCellString>{cell.getValue()}</TableCellString>;
				},
			}),
			columnHelper.accessor("server_id", {
				header: "Server",
				size: 30,
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
		data: recent ?? [],
		enableFilters: true,
		enableRowActions: true,
		state: {
			isLoading: isLoadingRecent || isLoadingServers,
			showAlertBanner: isErrorRecent || isErrorServers,
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

	return <SortableTable table={table} title={"Recent Speedruns"} />;
};

// const SpeedrunTopTable = ({
// 	speedruns,
// 	servers,
// 	isLoading,
// }: {
// 	speedruns: SpeedrunResult[];
// 	servers?: ServerSimple[];
// 	isLoading: boolean;
// }) => {
// 	const [pagination, setPagination] = useState({
// 		pageIndex: 0,
// 		pageSize: RowsPerPage.TwentyFive,
// 	});
// 	const keys = useMemo(() => {
// 	if (!speedruns) {
// 		return [];
// 	}
// 	return Object.keys(speedruns).sort();
// }, [speedruns]);
// 	const { data: speedruns, isLoading } = useQuery({
// 	queryKey: ["speedruns_overall"],
// 	queryFn: () => {
// 		return getSpeedrunsTopOverall(10);
// 	},
// });
//
// 	const columns = [
// 		columnHelper.accessor("rank", {
// 			header: "Rank",
// 			size: 10,
// 			cell: (info) => {
// 				const value = info.getValue();
// 				const weight = value <= 3 ? 700 : 500;
// 				return (
// 					<TableCell>
// 						<TextLink
// 							fontWeight={weight}
// 							to={"/speedruns/id/$speedrunId"}
// 							params={{ speedrunId: String(info.row.original.speedrun_id) }}
// 						>
// 							{value}
// 						</TextLink>
// 					</TableCell>
// 				);
// 			},
// 		}),
// 		columnHelper.accessor("duration", {
// 			header: "Time",
// 			size: 60,
// 			cell: (info) => (
// 				<TableCellSmall>
// 					<Typography align={"center"}>{durationString(info.getValue())}</Typography>
// 				</TableCellSmall>
// 			),
// 		}),
// 		columnHelper.accessor("players", {
// 			header: "Players",
// 			size: 30,
// 			cell: (info) => {
// 				return <TableCellString>{info.getValue().length}</TableCellString>;
// 			},
// 		}),
// 		columnHelper.accessor("server_id", {
// 			header: "Srv",
// 			size: 30,
// 			cell: (info) => {
// 				const srv = (servers ?? []).find((s) => (s.server_id = info.getValue()));
// 				return <TableCellString>{srv?.server_name}</TableCellString>;
// 			},
// 		}),
// 		columnHelper.accessor("created_on", {
// 			header: "Submitted",
// 			size: 100,
// 			cell: (info) => {
// 				return <TableCellString>{renderDateTime(info.getValue())}</TableCellString>;
// 			},
// 		}),
// 	];

// 	const opts: TableOptions<SpeedrunResult> = {
// 		data: speedruns,
// 		columns: columns,
// 		getCoreRowModel: getCoreRowModel(),
// 		manualPagination: false,
// 		autoResetPageIndex: true,
// 		onPaginationChange: setPagination,
// 		getPaginationRowModel: getPaginationRowModel(),
// 		state: { pagination },
// 	};

// 	const table = useReactTable(opts);

// 	return <DataTable table={table} isLoading={isLoading} />;
// };
