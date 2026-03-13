import type { MRT_RowData, MRT_TableOptions } from "material-react-table";

export const createDefaultTableOptions = <TData extends MRT_RowData>(): Partial<MRT_TableOptions<TData>> => ({
	enableGlobalFilter: false,
	enableRowPinning: false,
	initialState: { showColumnFilters: true, pagination: { pageSize: 25, pageIndex: 0 } },
	enableFacetedValues: true,
	enableColumnFilters: true,
	enableDensityToggle: false,
	enableTopToolbar: true,
	enableFilters: true,
	paginationDisplayMode: "pages",
	layoutMode: "grid",
	enableFullScreenToggle: true,
	positionActionsColumn: "last",
	positionToolbarAlertBanner: "top",
	columnFilterDisplayMode: "subheader",
	muiToolbarAlertBannerProps: () => ({
		color: "error",
		children: "Error loading data :(",
	}),
	muiTableBodyCellProps: {
		sx: { paddingLeft: 1, paddingRight: 1, paddingTop: 0.25, paddingBottom: 0.25 },
	},
	defaultColumn: {},
});
