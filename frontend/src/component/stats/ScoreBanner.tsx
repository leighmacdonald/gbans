import EmojiEventsIcon from "@mui/icons-material/EmojiEvents";
import Box from "@mui/material/Box";
import Paper from "@mui/material/Paper";
import Stack from "@mui/material/Stack";
import Typography from "@mui/material/Typography";
import { Team } from "../../rpc/stats/v1/stats_pb.ts";
import { blu, red, tf2Fonts } from "../../theme";

const ScoreSide = ({
	label,
	score,
	color,
	align,
	winner,
}: {
	label: string;
	score: number;
	color: string;
	align: "left" | "right";
	winner: boolean;
}) => {
	return (
		<Box
			sx={{
				flex: 1,
				backgroundColor: color,
				opacity: winner ? 1 : 0.75,
				padding: 1.5,
				display: "flex",
				flexDirection: "column",
				alignItems: align === "left" ? "flex-start" : "flex-end",
				borderBottom: winner ? "4px solid" : "4px solid transparent",
				borderBottomColor: winner ? "success.main" : "transparent",
			}}
		>
			<Stack direction="row" spacing={1} sx={{ alignItems: "center" }}>
				<Typography variant="h6" sx={{ ...tf2Fonts, color: "#fff" }}>
					{label}
				</Typography>
				{winner && <EmojiEventsIcon sx={{ color: "#ffd54f" }} fontSize="small" />}
			</Stack>
			<Typography variant="h1" sx={{ ...tf2Fonts, color: "#fff", lineHeight: 1 }}>
				{score}
			</Typography>
		</Box>
	);
};

export const ScoreBanner = ({ scoreBlu, scoreRed, winner }: { scoreBlu: number; scoreRed: number; winner: Team }) => {
	return (
		<Paper elevation={2}>
			<Stack direction="row">
				<ScoreSide label="BLU" score={scoreBlu} color={blu} align="left" winner={winner === Team.BLU} />
				<ScoreSide label="RED" score={scoreRed} color={red} align="right" winner={winner === Team.RED} />
			</Stack>
		</Paper>
	);
};
