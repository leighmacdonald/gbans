import Avatar from "@mui/material/Avatar";
import Badge from "@mui/material/Badge";
import { useTheme } from "@mui/material/styles";

export const ForumAvatar = ({ src, alt, online }: { src: string; alt: string; online: boolean }) => {
	const theme = useTheme();

	return (
		<Badge
			overlap={"circular"}
			anchorOrigin={{ vertical: "bottom", horizontal: "right" }}
			variant="dot"
			sx={{
				"& .MuiBadge-badge": {
					backgroundColor: online ? theme.palette.success.light : theme.palette.error.dark,
					color: online ? theme.palette.success.light : theme.palette.error.dark,
					boxShadow: `0 0 0 2px ${theme.palette.background.paper}`,
					"&::after": {
						position: "absolute",
						top: 0,
						left: 0,
						width: "100%",
						height: "100%",
						borderRadius: "50%",
						animation: online ? "ripple 1.2s infinite ease-in-out" : undefined,
						border: "1px solid currentColor",
						content: '""',
					},
				},
				"@keyframes ripple": {
					"0%": {
						transform: "scale(.8)",
						opacity: 1,
					},
					"100%": {
						transform: "scale(2.4)",
						opacity: 0,
					},
				},
			}}
		>
			<Avatar variant={"circular"} sx={{ height: "120px", width: "120px" }} src={src} alt={alt} />
		</Badge>
	);
};
