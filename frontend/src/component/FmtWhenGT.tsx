import Tooltip from "@mui/material/Tooltip";
import Typography from "@mui/material/Typography";

export const FmtWhenGt = (value: number, fmt?: (value: number) => string, gt: number = 0, fallback: string = "") => {
	return value > 1000 ? (
		<Tooltip title={`${value}`}>
			<Typography variant={"body1"} padding={0} sx={{ fontFamily: "Monospace" }}>
				{value > gt ? (fmt ? fmt(value) : `${value}`) : fallback}
			</Typography>
		</Tooltip>
	) : (
		<Typography variant={"body1"} padding={0} sx={{ fontFamily: "Monospace" }}>
			{value > gt ? (fmt ? fmt(value) : `${value}`) : fallback}
		</Typography>
	);
};
