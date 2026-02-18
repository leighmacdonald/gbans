import Badge, { type BadgeProps } from "@mui/material/Badge";
import { styled } from "@mui/material/styles";

export const StyledBadge = styled(Badge)<BadgeProps>(({ theme }) => ({
	"& .MuiBadge-badge": {
		right: -3,
		top: 13,
		border: `0px solid ${theme.palette.background.paper}`,
		padding: "0 2px",
		color: theme.palette.success.main,
		fontWeight: "bold",
		fontSize: "14px",
		backgroundColor: theme.palette.background.paper,
	},
}));
