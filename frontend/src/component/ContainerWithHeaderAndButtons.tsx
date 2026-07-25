import Grid from "@mui/material/Grid";
import Paper from "@mui/material/Paper";
import Stack from "@mui/material/Stack";
import { useTheme } from "@mui/material/styles";
import type { FC, JSX, ReactNode } from "react";
import { tf2Fonts } from "../theme";
import { VCenteredElement } from "./Heading";

interface ContainerWithHeaderProps {
	title: string;
	children?: JSX.Element[] | JSX.Element | string;
	iconLeft?: ReactNode;
	spacing?: number;
	elevation?: number;
	marginTop?: number;
	padding?: number;
	buttons?: ReactNode[];
}

export const ContainerWithHeaderAndButtons = ({
	title,
	children,
	iconLeft,
	spacing = 2,
	elevation = 1,
	marginTop = 0,
	padding = 1,
	buttons,
}: ContainerWithHeaderProps) => {
	return (
		<Paper elevation={elevation}>
			<HeadingWithButtons iconLeft={iconLeft} buttons={buttons}>
				{title}
			</HeadingWithButtons>
			<Stack spacing={spacing} sx={{ marginTop, padding }}>
				{children}
			</Stack>
		</Paper>
	);
};

interface HeadingWithButtonsProps {
	children: JSX.Element[] | JSX.Element | string;
	bgColor?: string;
	iconLeft?: ReactNode;
	buttons?: ReactNode[];
}

export const HeadingWithButtons: FC<HeadingWithButtonsProps> = ({
	children,
	bgColor,
	iconLeft,
	buttons,
}: HeadingWithButtonsProps) => {
	const theme = useTheme();
	return (
		<Grid
			container
			direction="row"
			//

			sx={{
				alignItems: "center",
				justifyContent: "flex-start",
				backgroundColor: bgColor ?? theme.palette.primary.main,
				color: theme.palette.common.white,
				padding: 1,
				...tf2Fonts,
			}}
		>
			{iconLeft && (
				<Grid size="auto" sx={{ paddingRight: 1 }}>
					<VCenteredElement icon={iconLeft} />
				</Grid>
			)}

			<Grid size="grow">{children}</Grid>
			{buttons && (
				<Grid size="auto" sx={{ paddingRight: 1 }}>
					{buttons}
				</Grid>
			)}
		</Grid>
	);
};
