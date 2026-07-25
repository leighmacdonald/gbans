import Paper from "@mui/material/Paper";
import Stack from "@mui/material/Stack";
import type { PropsWithChildren, ReactNode } from "react";
import { Heading } from "./Heading.tsx";

interface ContainerWithHeaderProps {
	title: ReactNode;
	iconLeft?: ReactNode;
	spacing?: number;
	elevation?: number;
	marginTop?: number;
	padding?: number;
}

export const ContainerWithHeader = ({
	title,
	children,
	iconLeft,
	spacing = 2,
	elevation = 1,
	marginTop = 0,
	padding = 0,
}: PropsWithChildren<ContainerWithHeaderProps>) => {
	return (
		<Paper elevation={elevation}>
			<Heading iconLeft={iconLeft}>{title}</Heading>
			<Stack
				spacing={spacing}
				sx={{
					padding,
					marginTop,
				}}
			>
				{children}
			</Stack>
		</Paper>
	);
};
