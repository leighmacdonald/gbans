import Box from "@mui/material/Box";
import Paper from "@mui/material/Paper";
import Stack from "@mui/material/Stack";
import Typography from "@mui/material/Typography";
import { Team } from "../../rpc/stats/v1/stats_pb.ts";
import { blu, red, tf2Fonts } from "../../theme";
import type { TeamTotals } from "./match";

const Stat = ({ label, value }: { label: string; value: number | string }) => (
	<Box sx={{ minWidth: 72 }}>
		<Typography variant="caption" color="textSecondary" sx={{ textTransform: "uppercase" }}>
			{label}
		</Typography>
		<Typography variant="h6" sx={{ lineHeight: 1.2 }}>
			{value.toLocaleString()}
		</Typography>
	</Box>
);

const TeamTotalsCard = ({ totals, color, label }: { totals?: TeamTotals; color: string; label: string }) => {
	if (!totals) {
		return null;
	}
	return (
		<Paper elevation={1} sx={{ flex: 1, borderTop: `4px solid ${color}` }}>
			<Box sx={{ padding: 1.5 }}>
				<Typography variant="subtitle1" sx={{ ...tf2Fonts }}>
					{label} Totals
				</Typography>
				<Stack direction="row" spacing={2} useFlexGap sx={{ flexWrap: "wrap", marginTop: 0.5 }}>
					<Stat label="Players" value={totals.players} />
					<Stat label="Kills" value={totals.kills} />
					<Stat label="Assists" value={totals.assists} />
					<Stat label="Deaths" value={totals.deaths} />
					<Stat label="K/D" value={(totals.kills / Math.max(1, totals.deaths)).toFixed(1)} />
					<Stat label="Damage" value={totals.damage} />
					<Stat label="Healing" value={totals.healing} />
					<Stat label="Charges" value={totals.charges} />
					<Stat label="Drops" value={totals.drops} />
					<Stat label="Caps" value={totals.captures} />
				</Stack>
			</Box>
		</Paper>
	);
};

export const TeamTotalsStrip = ({ totals }: { totals: TeamTotals[] }) => {
	const bluTotals = totals.find((t) => t.team === Team.BLU);
	const redTotals = totals.find((t) => t.team === Team.RED);
	return (
		<Stack direction={{ xs: "column", md: "row" }} spacing={2}>
			<TeamTotalsCard totals={bluTotals} color={blu} label="BLU" />
			<TeamTotalsCard totals={redTotals} color={red} label="RED" />
		</Stack>
	);
};
