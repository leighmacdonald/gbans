import Box from "@mui/material/Box";
import Grid from "@mui/material/Grid";
import Paper from "@mui/material/Paper";
import Typography from "@mui/material/Typography";
import {
	type MRT_RowData,
	MRT_ShowHideColumnsButton,
	MRT_TableContainer,
	type MRT_TableInstance,
	MRT_TablePagination,
	MRT_ToggleFiltersButton,
	MRT_ToolbarAlertBanner,
} from "material-react-table";
import type { ReactNode } from "react";
import { emptyOrNullString } from "../../util/types";
import { VCenteredElement } from "../Heading";

type Props<TData extends MRT_RowData> = {
	table: MRT_TableInstance<TData>;
	title?: string;
	buttons?: ReactNode[];
	hideToolbarButtons?: boolean;
	hideHeader?: boolean;
	hidePagination?: boolean;
	unknownRowCount?: boolean;
	iconLeft?: ReactNode;
};

export const SortableTable = <TData extends MRT_RowData>({
	table,
	title,
	buttons,
	iconLeft,
	hideHeader = false,
	hideToolbarButtons = false,
	hidePagination = false,
	unknownRowCount = false,
}: Props<TData>) => {
	return (
		<Paper>
			{!hideHeader && (
				<Grid
					container
					direction={"row"}
					spacing={1}
					sx={() => ({
						backgroundColor: "primary.main",
						borderRadius: "4px 4px 0 0",
						borderRadiusBottom: 0,
						padding: 1,
						"@media (max-width: 768px)": {
							flexDirection: "column",
						},
					})}
				>
					<Grid sx={{ padding: 1, paddingRight: 0 }}>
						<VCenteredElement icon={iconLeft} />
					</Grid>
					{!emptyOrNullString(title) && (
						<Grid>
							<Typography
								variant="h6"
								sx={{
									padding: 1,
									display: "inline-block",
									fontWeight: 900,
									color: "white",
								}}
							>
								{title}
							</Typography>
						</Grid>
					)}
					{buttons && <Grid>{buttons}</Grid>}
					<Grid sx={{ marginLeft: "auto" }}>
						<Box sx={{ display: "flex", alignItems: "center", gap: 1 }}>
							{!hideToolbarButtons && (
								<>
									<MRT_ShowHideColumnsButton table={table} sx={{ color: "primary.contrastText" }} />
									<MRT_ToggleFiltersButton table={table} sx={{ color: "primary.contrastText" }} />
									{/*<MRT_ToggleDensePaddingButton table={table} sx={{ color: "primary.contrastText" }} />*/}
									{/*<MRT_ToggleFullScreenButton table={table} sx={{ color: "primary.contrastText" }} />*/}
								</>
							)}
						</Box>
					</Grid>
				</Grid>
			)}

			<Box sx={{ display: "grid", width: "100%" }}>
				<MRT_ToolbarAlertBanner stackAlertBanner table={table} />
			</Box>

			<MRT_TableContainer table={table} />
			{!hidePagination && (
				<Box>
					<Box sx={{ display: "flex", justifyContent: "flex-end" }}>
						{unknownRowCount ? (
							<MRT_TablePagination table={table} />
						) : (
							<MRT_TablePagination table={table} />
						)}
					</Box>
				</Box>
			)}
		</Paper>
	);
};
