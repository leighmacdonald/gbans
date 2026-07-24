import Box from "@mui/material/Box";
import Grid from "@mui/material/Grid";
import { useTheme } from "@mui/material/styles";
import type { PropsWithChildren, ReactNode } from "react";
import { tf2Fonts } from "../theme";

interface HeadingProps {
	bgColor?: string;
	iconLeft?: ReactNode;
	iconRight?: ReactNode;
	align?: "flex-start" | "center" | "flex-end" | "space-between";
}

interface VCenteredImageProps {
	icon: ReactNode;
}

export const VCenteredElement = ({ icon }: VCenteredImageProps) => {
	return <Box sx={{ display: "flex", justifyContent: "right", alignItems: "center" }}>{icon}</Box>;
};

export const Heading = ({ children, bgColor, iconLeft, iconRight, align }: PropsWithChildren<HeadingProps>) => {
	const theme = useTheme();
	return (
		<Grid
			container
			direction="row"
			sx={{
				backgroundColor: bgColor ?? theme.palette.primary.main,
				color: theme.palette.common.white,
				alignItems: "center",
				justifyContent: align ?? "flex-start",
				padding: 1,
				...tf2Fonts,
			}}
		>
			{iconLeft && (
				<Grid
					sx={{
						paddingRight: 1,
					}}
				>
					<VCenteredElement icon={iconLeft} />
				</Grid>
			)}
			<Grid>{children}</Grid>
			{iconRight && (
				<Grid>
					<VCenteredElement icon={iconRight} />
				</Grid>
			)}
		</Grid>
	);
};
