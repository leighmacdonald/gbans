import Box from "@mui/material/Box";
import Typography from "@mui/material/Typography";

interface InfoBarProps {
	title: string;
	value: string | number;
	align?: "left" | "right";
}

export const InfoBar = ({ title, value, align = "left" }: InfoBarProps) => {
	return (
		<Box>
			<Typography variant={"subtitle1"} fontWeight={500} align={align}>
				{title}
			</Typography>
			<Typography variant={"h3"} fontWeight={700} align={align}>
				{value}
			</Typography>
		</Box>
	);
};
