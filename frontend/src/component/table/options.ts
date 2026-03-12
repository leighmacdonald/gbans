import type { MRT_RowData, MRT_TableOptions } from "material-react-table";

export const createDefaultTableOptions = <TData extends MRT_RowData>(): Partial<MRT_TableOptions<TData>> => ({
	enableGlobalFilter: false,
	enableRowPinning: false,
	initialState: { showColumnFilters: true },
	enableFacetedValues: true,
	enableColumnFilters: true,
	manualFiltering: undefined,
	manualPagination: undefined,
	manualSorting: undefined,
	enableDensityToggle: false,
	enableTopToolbar: true,
	paginationDisplayMode: "pages",
	layoutMode: "grid",
	enableFullScreenToggle: true,
	positionActionsColumn: "last",
	columnFilterDisplayMode: "subheader",

	muiTableBodyCellProps: {
		sx: { paddingLeft: 1, paddingRight: 1, paddingTop: 0.25, paddingBottom: 0.25 },
	},
	defaultColumn: {},
});
