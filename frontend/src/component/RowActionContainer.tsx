import Box from "@mui/material/Box";

export const RowActionContainer = ({ children }: { children: React.ReactNode }) => (
	<Box sx={{ display: "flex", flexWrap: "nowrap", gap: "8px" }}>{children}</Box>
);
