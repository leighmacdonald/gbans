import { Typography } from "@mui/material";
import type { PropsWithChildren } from "react";

export const SubHeading = ({ children }: PropsWithChildren) => (
	<Typography variant={"subtitle1"} padding={1}>
		{children}
	</Typography>
);
