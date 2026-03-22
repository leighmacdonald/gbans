import { Box } from "@mui/material";

export const RowActionContainer = ({ children }: { children: React.ReactNode }) => (
	<Box sx={{ display: "flex", flexWrap: "nowrap", gap: "8px" }}>{children}</Box>
);
