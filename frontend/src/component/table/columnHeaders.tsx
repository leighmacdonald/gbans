import Tooltip from "@mui/material/Tooltip";
import Typography from "@mui/material/Typography";

/**
 * Abbreviated MRT column header with the full description in a hover tooltip.
 */
export const shortHeader = (short: string, full: string) => ({
	header: short,
	Header: () => (
		<Tooltip title={full}>
			<Typography variant="body2" sx={{ fontWeight: 700 }}>
				{short}
			</Typography>
		</Tooltip>
	),
});
