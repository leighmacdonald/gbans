import { useTheme } from "@mui/material";
import Typography from "@mui/material/Typography";
import Stack from "@mui/system/Stack";
import type { ReactNode } from "react";
import { tf2Fonts } from "../theme";

export const NewsHead = ({ left, right }: { left: ReactNode; right: ReactNode }) => {
	const theme = useTheme();
	return (
		<Stack
			spacing={1}
			padding={1}
			paddingLeft={2}
			paddingRight={2}
			direction={"row"}
			sx={{ backgroundColor: theme.palette.primary.main }}
		>
			<Typography
				lineHeight={2}
				variant={"subtitle1"}
				align={"left"}
				sx={{
					color: theme.palette.common.white,
					width: "100%",
					...tf2Fonts,
				}}
			>
				{left}
			</Typography>

			<Typography
				lineHeight={2}
				variant={"subtitle1"}
				align={"right"}
				sx={{
					...tf2Fonts,
				}}
			>
				{right}
			</Typography>
		</Stack>
	);
};
