import type { MRT_RowData, MRT_TableOptions } from "material-react-table";

export const createDefaultTableOptions = <TData extends MRT_RowData>(): Partial<MRT_TableOptions<TData>> => ({
	enableGlobalFilter: false,
	enableRowPinning: false,
	initialState: { showColumnFilters: true },
	manualFiltering: undefined,
	manualPagination: undefined,
	manualSorting: undefined,
	enableDensityToggle: false,
	enableTopToolbar: false,
	paginationDisplayMode: "pages",
	layoutMode: "grid",
	enableFullScreenToggle: false,
	positionActionsColumn: "last",
	columnFilterDisplayMode: "popover",
	muiTablePaperProps: {
		elevation: 0,
	},
	muiTableBodyCellProps: {
		sx: { paddingLeft: 1, paddingRight: 1, paddingTop: 0.25, paddingBottom: 0.25 },
	},
	defaultColumn: {
		//you can even list default column options here
	},
});
