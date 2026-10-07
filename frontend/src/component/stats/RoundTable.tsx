import Chip from "@mui/material/Chip";
import Table from "@mui/material/Table";
import TableBody from "@mui/material/TableBody";
import TableCell from "@mui/material/TableCell";
import TableHead from "@mui/material/TableHead";
import TableRow from "@mui/material/TableRow";
import Tooltip from "@mui/material/Tooltip";
import Typography from "@mui/material/Typography";
import { createMRTColumnHelper, useMaterialReactTable } from "material-react-table";
import { useMemo } from "react";
import { type RoundPlayer, Team } from "../../rpc/stats/v1/stats_pb.ts";
import { blu, red } from "../../theme.ts";
import { durationString } from "../../util/time.ts";
import { PersonCell } from "../PersonCell.tsx";
import { shortHeader } from "../table/columnHeaders";
import { createDefaultTableOptions } from "../table/options";
import { SortableTable } from "../table/SortableTable";
import type { MatchRound } from "./match";

const defaultRoundOptions = createDefaultTableOptions<MatchRound>();
const roundColumnHelper = createMRTColumnHelper<MatchRound>();

const winnerLabel = (winner: Team): string => {
	switch (winner) {
		case Team.RED:
			return "RED";
		case Team.BLU:
			return "BLU";
		default:
			return "—";
	}
};

const winnerColor = (winner: Team): string | undefined => {
	switch (winner) {
		case Team.RED:
			return red;
		case Team.BLU:
			return blu;
		default:
			return undefined;
	}
};

const RoundPlayersDetail = ({ players }: { players: RoundPlayer[] }) => {
	const rows = useMemo(
		() => [...players].toSorted((a, b) => Number(b.scoreboardDamage) - Number(a.scoreboardDamage)),
		[players],
	);
	return (
		<Table size="small">
			<TableHead>
				<TableRow>
					<TableCell>
						<Tooltip title="Player">
							<span>Player</span>
						</Tooltip>
					</TableCell>
					<TableCell>
						<Tooltip title="Team">
							<span>Team</span>
						</Tooltip>
					</TableCell>
					<TableCell align="right">
						<Tooltip title="Kills">
							<span>K</span>
						</Tooltip>
					</TableCell>
					<TableCell align="right">
						<Tooltip title="Assists">
							<span>A</span>
						</Tooltip>
					</TableCell>
					<TableCell align="right">
						<Tooltip title="Deaths">
							<span>D</span>
						</Tooltip>
					</TableCell>
					<TableCell align="right">
						<Tooltip title="Damage dealt">
							<span>Damage</span>
						</Tooltip>
					</TableCell>
					<TableCell align="right">
						<Tooltip title="Healing">
							<span>Healing</span>
						</Tooltip>
					</TableCell>
				</TableRow>
			</TableHead>
			<TableBody>
				{rows.map((p) => (
					<TableRow key={p.person?.steamId ?? `${p.roundId}`}>
						<TableCell>
							{p.person ? (
								<PersonCell
									steamId={p.person.steamId}
									avatarHash={p.person.avatarHash}
									personaName={p.person.name}
								/>
							) : (
								<Typography variant="body2">Unknown</Typography>
							)}
						</TableCell>
						<TableCell>{p.team === Team.RED ? "RED" : p.team === Team.BLU ? "BLU" : "—"}</TableCell>
						<TableCell align="right">{Number(p.kills)}</TableCell>
						<TableCell align="right">{Number(p.assists)}</TableCell>
						<TableCell align="right">{Number(p.scoreboardDeaths)}</TableCell>
						<TableCell align="right">{Number(p.scoreboardDamage).toLocaleString()}</TableCell>
						<TableCell align="right">{Number(p.healing).toLocaleString()}</TableCell>
					</TableRow>
				))}
			</TableBody>
		</Table>
	);
};

export const RoundTable = ({ data }: { data: MatchRound[] }) => {
	const columns = useMemo(
		() => [
			roundColumnHelper.accessor("winner", {
				grow: false,
				enableSorting: false,
				...shortHeader("Winner", "Round winner"),
				size: 90,
				Cell: ({ row }) => (
					<Chip
						label={winnerLabel(row.original.winner)}
						size="small"
						sx={{
							backgroundColor: winnerColor(row.original.winner),
							color: "#fff",
							fontWeight: 700,
							minWidth: 56,
						}}
					/>
				),
			}),
			roundColumnHelper.accessor("durationMs", {
				grow: false,
				enableSorting: false,
				...shortHeader("Length", "Round length"),
				size: 80,
				Cell: ({ cell }) => durationString(Number(cell.getValue()) * 1000),
			}),
			roundColumnHelper.display({
				id: "score",
				grow: false,
				...shortHeader("Score", "Match score after this round (BLU – RED)"),
				size: 80,
				Cell: ({ row }) => (
					<Typography variant="body2" sx={{ fontWeight: 700 }}>
						{row.original.scoreBlu} – {row.original.scoreRed}
					</Typography>
				),
			}),
			roundColumnHelper.accessor("killsBlu", {
				grow: false,
				enableSorting: false,
				...shortHeader("BLU K", "BLU kills"),
				size: 70,
			}),
			roundColumnHelper.accessor("killsRed", {
				grow: false,
				enableSorting: false,
				...shortHeader("RED K", "RED kills"),
				size: 70,
			}),
			roundColumnHelper.accessor("chargesBlu", {
				grow: false,
				enableSorting: false,
				...shortHeader("BLU UC", "BLU ÜberCharges"),
				size: 70,
			}),
			roundColumnHelper.accessor("chargesRed", {
				grow: false,
				enableSorting: false,
				...shortHeader("RED UC", "RED ÜberCharges"),
				size: 70,
			}),
			roundColumnHelper.accessor("damageBlu", {
				grow: false,
				enableSorting: false,
				...shortHeader("BLU DA", "BLU damage dealt"),
				size: 80,
				Cell: ({ cell }) => Number(cell.getValue()).toLocaleString(),
			}),
			roundColumnHelper.accessor("damageRed", {
				grow: false,
				enableSorting: false,
				...shortHeader("RED DA", "RED damage dealt"),
				size: 80,
				Cell: ({ cell }) => Number(cell.getValue()).toLocaleString(),
			}),
			roundColumnHelper.display({
				id: "status",
				grow: false,
				...shortHeader("Status", "Round outcome"),
				Cell: ({ row }) => {
					return row.original.isStalemate
						? "Stalemate"
						: row.original.isSuddenDeath
							? "Sudden Death"
							: "Decided";
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
		renderDetailPanel: ({ row }) => <RoundPlayersDetail players={row.original.players} />,
		initialState: {
			...defaultRoundOptions.initialState,
			columnVisibility: {
				winner: true,
			},
		},
	});

	return <SortableTable table={roundTable} title={"Rounds"} hidePagination={true} />;
};
