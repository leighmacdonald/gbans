import Typography from "@mui/material/Typography";
import type { PropsWithChildren } from "react";

export const SubHeading = ({ children }: PropsWithChildren) => (
	<Typography
		variant={"subtitle1"}
		sx={{
			padding: 1,
		}}
	>
		{children}
	</Typography>
);
