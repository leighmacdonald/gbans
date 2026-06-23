import Typography from "@mui/material/Typography";
import { Stack } from "@mui/system";
import { createMRTColumnHelper, useMaterialReactTable } from "material-react-table";
import { useMemo } from "react";
import { Team } from "../../rpc/stats/v1/stats_pb.ts";
import { blu, red } from "../../theme.ts";
import { durationString } from "../../util/time.ts";
import { PersonCell } from "../PersonCell.tsx";
import { createDefaultTableOptions } from "../table/options";
import { SortableTable } from "../table/SortableTable";
import type { MatchRound } from "./match";

const defaultRoundOptions = createDefaultTableOptions<MatchRound>();
const roundColumnHelper = createMRTColumnHelper<MatchRound>();

export const RoundTable = ({ data }: { data: MatchRound[] }) => {
	const columns = useMemo(
		() => [
			roundColumnHelper.accessor("winner", {
				grow: false,
				enableSorting: false,
				header: "Winner",
				Cell: ({ row }) => {
					const colour =
						row.original.winner === Team.RED ? red : row.original.winner === Team.BLU ? blu : undefined;
					return (
						<Typography bgcolor={colour} width={"100%"} textAlign={"center"} fontFamily={"TF2 Build"}>
							{row.original.winner === Team.RED ? "RED" : row.original.winner === Team.BLU ? "BLU" : ""}
						</Typography>
					);
				},
			}),
			roundColumnHelper.accessor("durationMs", {
				grow: false,
				enableSorting: false,
				header: "Duration",
				Cell: ({ cell }) => durationString(Number(cell.getValue()) * 1000),
			}),
			roundColumnHelper.display({
				id: "status",
				grow: false,
				header: "Status",
				Cell: ({ row }) => {
					return row.original.isStalemate ? "Stalemate" : row.original.isSuddenDeath ? "Sudden Death" : "Won";
				},
			}),
			roundColumnHelper.display({
				grow: true,
				header: "Most Valuable Players",
				Cell: ({ row }) => {
					const mvps = row.original.players
						.filter((p) => p.mvp && p.person)
						.toSorted((a, b) => (a.points < b.points ? 1 : -1))
						.map((p) => p.person);

					return (
						<Stack direction={"row"}>
							{mvps.map((p) => {
								if (!p) {
									return null;
								}
								return (
									<PersonCell
										key={p.steamId}
										steamId={p.steamId}
										avatarHash={p.avatarHash}
										personaName={p.name}
									/>
								);
							})}
						</Stack>
					);
				},
			}),
		],
		[],
	);

	const roundTable = useMaterialReactTable({
		...defaultRoundOptions,
		enableRowNumbers: true,
		columns,
		data,
		enableFilters: false,
		enableFacetedValues: false,
		enableColumnActions: false,
		enablePagination: false,
		initialState: {
			...defaultRoundOptions.initialState,
			columnVisibility: {
				winner: true,
			},
		},
	});

	return <SortableTable table={roundTable} title={"Rounds"} hidePagination={true} />;
};
