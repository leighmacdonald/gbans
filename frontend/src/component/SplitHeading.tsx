import Stack from "@mui/material/Stack";
import { useTheme } from "@mui/material/styles";
import Typography from "@mui/material/Typography";
import type { FC } from "react";
import { tf2Fonts } from "../theme";

interface SplitHeadingProps {
	left: string;
	right: string;
	bgColor?: string;
}

export const SplitHeading: FC<SplitHeadingProps> = ({ left, right, bgColor }: SplitHeadingProps) => {
	const theme = useTheme();
	return (
		<Stack direction={"row"}>
			<Typography
				lineHeight={2}
				variant={"subtitle1"}
				align={"left"}
				paddingLeft={2}
				paddingTop={1}
				paddingBottom={1}
				sx={{
					backgroundColor: bgColor ?? theme.palette.primary.main,
					color: theme.palette.common.white,
					width: "100%",
					...tf2Fonts,
				}}
			>
				{left}
			</Typography>
			<Typography
				variant={"subtitle1"}
				align={"right"}
				paddingTop={1}
				paddingBottom={1}
				paddingRight={2}
				sx={{
					backgroundColor: bgColor ?? theme.palette.primary.main,
					color: theme.palette.common.white,
					width: 200,
				}}
			>
				{right}
			</Typography>
		</Stack>
	);
};
