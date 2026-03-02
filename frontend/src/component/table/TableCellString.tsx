import type { TypographyVariant } from "@mui/material";
import Typography from "@mui/material/Typography";
import type { ElementType, PropsWithChildren } from "react";

interface TextProps {
	variant?: TypographyVariant;
	component?: ElementType;
	onClick?: () => void;
	title?: string;
}

export const TableCellString = ({
	children,
	variant = "body1",
	component = "p",
	title = undefined,
}: PropsWithChildren<TextProps>) => {
	return (
		<div title={title}>
			<Typography variant={variant} component={component}>
				{children}
			</Typography>
		</div>
	);
};
