import type { MRT_RowData, MRT_TableOptions } from "material-react-table";
import z from "zod/v4";

export const createDefaultTableOptions = <TData extends MRT_RowData>(): Partial<MRT_TableOptions<TData>> => ({
	enableGlobalFilter: false,
	enableRowPinning: false,
	initialState: { showColumnFilters: true },
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
	// muiToolbarAlertBannerProps: () => ({
	// 	color: "error",
	// 	children: "Error loading data :(",
	// }),
	muiTableBodyCellProps: {
		sx: { paddingLeft: 1, paddingRight: 1, paddingTop: 0.25, paddingBottom: 0.25 },
	},
	defaultColumn: {},
});

export type Updater<T> = T | ((old: T) => T);
export type OnChangeFn<T> = (updaterOrValue: Updater<T>) => void;

export const makeSchemaState = ({
	defaultSortColumn,
	defaultDesc = true,
}: {
	defaultSortColumn: string;
	defaultDesc?: boolean;
}) => {
	return z.object({
		pagination: z
			.object({
				pageIndex: z.number().positive().catch(0),
				pageSize: z.number().positive().catch(10),
			})
			.default({ pageIndex: 0, pageSize: 50 })
			.optional(),
		columnFilters: z
			.object({
				id: z.string(),
				value: z.unknown(),
			})
			.array()
			.default([])
			.optional(),
		sorting: z
			.object({
				id: z.string(),
				desc: z.boolean().catch(true),
			})
			.array()
			.default([
				{
					id: defaultSortColumn,
					desc: Boolean(defaultDesc),
				},
			])
			.optional(),
	}).shape;
};
