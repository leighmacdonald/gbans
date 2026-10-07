import Chip from "@mui/material/Chip";
import Table from "@mui/material/Table";
import TableBody from "@mui/material/TableBody";
import TableCell from "@mui/material/TableCell";
import TableContainer from "@mui/material/TableContainer";
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

const H = ({ title, short, align }: { title: string; short: string; align?: "right" }) => (
	<TableCell align={align}>
		<Tooltip title={title}>
			<span>{short}</span>
		</Tooltip>
	</TableCell>
);

const fmt1dp = (value: number): string => (Number.isFinite(value) ? value.toFixed(1) : "0.0");

const chargesOf = (p: RoundPlayer): number =>
	Number(p.chargesUber) + Number(p.chargesKritz) + Number(p.chargesVacc) + Number(p.chargesQuickfix);

const RoundPlayersDetail = ({ players, durationMs }: { players: RoundPlayer[]; durationMs: number }) => {
	const rows = useMemo(
		() => [...players].toSorted((a, b) => Number(b.scoreboardDamage) - Number(a.scoreboardDamage)),
		[players],
	);
	const durationMins = Math.max(1, Number(durationMs)) / 60000;
	return (
		<TableContainer sx={{ maxWidth: "100%", overflowX: "auto" }}>
			<Table size="small">
				<TableHead>
					<TableRow>
						<H title="Player" short="Player" />
						<H title="Team" short="Team" />
						<H title="Kills" short="K" align="right" />
						<H title="Assists" short="A" align="right" />
						<H title="Deaths" short="D" align="right" />
						<H title="Damage dealt" short="DA" align="right" />
						<H title="Damage dealt per minute" short="DA/M" align="right" />
						<H title="(Kills + Assists) / Deaths" short="KA/D" align="right" />
						<H title="Kills / Deaths" short="K/D" align="right" />
						<H title="Damage taken" short="DT" align="right" />
						<H title="Damage taken per minute" short="DT/M" align="right" />
						<H title="Healing" short="HP" align="right" />
						<H title="ÜberCharges (all types)" short="UC" align="right" />
						<H title="Charge drops" short="Drops" align="right" />
						<H title="Airshots" short="AS" align="right" />
						<H title="Backstabs" short="BS" align="right" />
						<H title="Headshots" short="HS" align="right" />
						<H title="Point captures" short="CAP" align="right" />
						<H title="Captures blocked" short="Blocked" align="right" />
						<H title="Scoreboard points" short="P" align="right" />
					</TableRow>
				</TableHead>
				<TableBody>
					{rows.map((p) => {
						const kills = Number(p.kills);
						const assists = Number(p.assists);
						const deaths = Number(p.scoreboardDeaths);
						const damage = Number(p.scoreboardDamage);
						const damageTaken = Number(p.damageTaken);
						return (
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
								<TableCell align="right">{kills}</TableCell>
								<TableCell align="right">{assists}</TableCell>
								<TableCell align="right">{deaths}</TableCell>
								<TableCell align="right">{damage.toLocaleString()}</TableCell>
								<TableCell align="right">
									{Math.round(damage / durationMins).toLocaleString()}
								</TableCell>
								<TableCell align="right">{fmt1dp((kills + assists) / Math.max(1, deaths))}</TableCell>
								<TableCell align="right">{fmt1dp(kills / Math.max(1, deaths))}</TableCell>
								<TableCell align="right">{damageTaken.toLocaleString()}</TableCell>
								<TableCell align="right">
									{Math.round(damageTaken / durationMins).toLocaleString()}
								</TableCell>
								<TableCell align="right">{Number(p.healing).toLocaleString()}</TableCell>
								<TableCell align="right">{chargesOf(p)}</TableCell>
								<TableCell align="right">{Number(p.drops)}</TableCell>
								<TableCell align="right">{Number(p.airshots)}</TableCell>
								<TableCell align="right">{Number(p.backstabs)}</TableCell>
								<TableCell align="right">{Number(p.headshots)}</TableCell>
								<TableCell align="right">{Number(p.captures)}</TableCell>
								<TableCell align="right">{Number(p.capturesBlocked)}</TableCell>
								<TableCell align="right">{Number(p.points).toLocaleString()}</TableCell>
							</TableRow>
						);
					})}
				</TableBody>
			</Table>
		</TableContainer>
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
		renderDetailPanel: ({ row }) => (
			<RoundPlayersDetail players={row.original.players} durationMs={row.original.durationMs} />
		),
		initialState: {
			...defaultRoundOptions.initialState,
			columnVisibility: {
				winner: true,
			},
		},
	});

	return <SortableTable table={roundTable} title={"Rounds"} hidePagination={true} />;
};
