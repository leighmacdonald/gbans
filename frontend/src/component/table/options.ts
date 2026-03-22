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
	// paginationDisplayMode: "pages",
	layoutMode: "grid",
	displayColumnDefOptions: makeRowActionsDefOptions(),
	enableFullScreenToggle: true,
	positionActionsColumn: "first",
	positionToolbarAlertBanner: "top",
	columnFilterDisplayMode: "subheader",
	// muiToolbarAlertBannerProps: () => ({
	// 	color: "error",
	// 	children: "Error loading data :(",
	// }),
	muiTableBodyCellProps: {
		sx: { paddingLeft: 1, paddingRight: 1, paddingTop: 0.25, paddingBottom: 0.25 },
	},
});

export type Updater<T> = T | ((old: T) => T);
export type OnChangeFn<T> = (updaterOrValue: Updater<T>) => void;

export const dateTimeColumnSize = 150;

export const makeRowActionsDefOptions = (count: number = 1) => {
	return {
		"mrt-row-actions": {
			size: 60 * count, //if using layoutMode that is not 'semantic', the columns will not auto-size, so you need to set the size manually
			grow: false,
			header: "",
		},
	};
};

export const makeSchemaState = (defaultSortColumn: string = "", defaultDesc: boolean = true) => {
	return z.object({
		pagination: z
			.object({
				pageIndex: z.number().positive().catch(0),
				pageSize: z.number().positive().catch(25),
			})
			.default({ pageIndex: 0, pageSize: 25 })
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

export const filterValueNumberArray = <T, V extends number>(id: keyof T, s?: MRT_ColumnFiltersState): V[] =>
	(s?.find((filter) => filter.id === id)?.value as V[]) ?? ([] as V[]);

export const filterValueBool = <T>(id: keyof T, s?: MRT_ColumnFiltersState): boolean =>
	Boolean(s?.find((filter) => filter.id === id)?.value ?? false);

export const filterValueDefault = <T>(id: keyof T, defaultValue?: unknown, s?: MRT_ColumnFiltersState) =>
	filterValue(id, s) ?? defaultValue;

export const sortValueDefault = (sorting: MRT_SortingState, id: string, desc: boolean = true) =>
	sorting?.find((sort) => sort) ?? { id, desc };

export const filterValueDate = <T>(id: keyof T, s?: MRT_ColumnFiltersState): Date | string => {
	try {
		const found = s?.find((filter) => filter.id === id)?.value;
		if (!found) return "";
		const d = new Date(String(found));
		if (d instanceof Date && !Number.isNaN(d.getTime())) {
			return d;
		}
		return "";
	} catch {
		return "";
	}
};
