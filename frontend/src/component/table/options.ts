import type { MRT_ColumnFiltersState, MRT_RowData, MRT_SortingState, MRT_TableOptions } from "material-react-table";
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

export const makeSchemaState = (defaultSortColumn: string = "", defaultDesc: boolean = true) => {
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
			.default(
				defaultSortColumn !== ""
					? [
							{
								id: defaultSortColumn,
								desc: Boolean(defaultDesc),
							},
						]
					: [],
			)
			.optional(),
	});
};
export const defaultSearchSchema = makeSchemaState();

type SearchSchema = z.infer<typeof defaultSearchSchema>;

export const setColumnFilter = (search: SearchSchema, id: string, value: unknown) => {
	const columnFilters = [...(search.columnFilters ?? []).filter((f) => f.id !== id), { id, value }];
	return {
		...search,
		columnFilters,
	};
};

export const filterValue = <T>(id: keyof T, s?: MRT_ColumnFiltersState): string =>
	String(s?.find((filter) => filter.id === id)?.value ?? "");

export const filterValueNumber = <T>(id: keyof T, s?: MRT_ColumnFiltersState): number =>
	Number(s?.find((filter) => filter.id === id)?.value ?? 0);

export const filterValueBool = <T>(id: keyof T, s?: MRT_ColumnFiltersState): boolean =>
	Boolean(s?.find((filter) => filter.id === id)?.value ?? false);

export const filterValueDefault = <T>(id: keyof T, defaultValue?: unknown, s?: MRT_ColumnFiltersState) =>
	filterValue(id, s) ?? defaultValue;

export const sortValueDefault = (sorting: MRT_SortingState, id: string, desc: boolean = true) =>
	sorting?.find((sort) => sort) ?? { id, desc };
