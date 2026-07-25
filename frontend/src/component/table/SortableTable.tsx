import Box from "@mui/material/Box";
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

type Props<TData extends MRT_RowData> = {
	table: MRT_TableInstance<TData>;
	title?: string;
	buttons?: ReactNode[];
	hideToolbarButtons?: boolean;
	hideHeader?: boolean;
	hidePagination?: boolean;
	unknownRowCount?: boolean;
};

export const SortableTable = <TData extends MRT_RowData>({
	table,
	title,
	buttons,
	hideHeader = false,
	hideToolbarButtons = false,
	hidePagination = false,
	unknownRowCount = false,
}: Props<TData>) => {
	return (
		<Paper>
			{!hideHeader && (
				<Box
					sx={() => ({
						display: "flex",
						backgroundColor: "primary.main",
						borderRadius: "4px 4px 0 0",
						borderRadiusBottom: 0,
						flexDirection: "row",
						gap: 2,
						justifyContent: "space-between",
						padding: 1,
						"@media (max-width: 768px)": {
							flexDirection: "column",
						},
					})}
				>
					{!emptyOrNullString(title) && (
						<Box>
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
							{buttons}
						</Box>
					)}
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
				</Box>
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
