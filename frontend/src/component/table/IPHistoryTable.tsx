import Typography from "@mui/material/Typography";
import { useQuery } from "@tanstack/react-query";
import {
	createMRTColumnHelper,
	type MRT_ColumnFiltersState,
	type MRT_PaginationState,
	type MRT_SortingState,
	useMaterialReactTable,
} from "material-react-table";
import { useMemo, useState } from "react";
import { apiGetConnections } from "../../api/profile.ts";
import type { PersonConnection } from "../../schema/people.ts";
import { renderDateTime } from "../../util/time.ts";
import { createDefaultTableOptions } from "./options.ts";
import { SortableTable } from "./SortableTable.tsx";

const columnHelper = createMRTColumnHelper<PersonConnection>();
const defaultOptions = createDefaultTableOptions<PersonConnection>();

export const IPHistoryTable = ({ steamId }: { steamId: string }) => {
	const [columnFilters, setColumnFilters] = useState<MRT_ColumnFiltersState>([]);
	const [globalFilter, setGlobalFilter] = useState("");
	const [sorting, setSorting] = useState<MRT_SortingState>([]);
	const [pagination, setPagination] = useState<MRT_PaginationState>({
		pageIndex: 0,
		pageSize: 50,
	});

	const { data, isLoading, isError } = useQuery({
		queryKey: ["connectionHist", { columnFilters, globalFilter, pagination, sorting }],
		queryFn: async ({ signal }) => {
			const sort = sorting.find((sort) => sort);
			return await apiGetConnections(
				{
					limit: pagination.pageSize,
					offset: pagination.pageIndex * pagination.pageSize,
					order_by: sort ? sort.id : "created_on",
					desc: sort ? sort.desc : false,
					source_id: steamId,
				},
				signal,
			);
		},
	});
	const columns = useMemo(() => {
		return [
			columnHelper.accessor("created_on", {
				header: "Created",
				size: 120,
				Cell: ({ cell }) => <Typography>{renderDateTime(cell.getValue())}</Typography>,
			}),
			columnHelper.accessor("persona_name", {
				header: "Name",
				Cell: ({ cell }) => <Typography>{cell.getValue()}</Typography>,
			}),
			columnHelper.accessor("ip_addr", {
				header: "IP Address",
				size: 120,
				Cell: ({ cell }) => <Typography>{cell.getValue()}</Typography>,
			}),
			columnHelper.accessor("server_id", {
				header: "Server",
				size: 120,
				Cell: ({ row }) => <Typography>{row.original.server_name_short}</Typography>,
			}),
		];
	}, []);

	const table = useMaterialReactTable({
		...defaultOptions,
		columns,
		data: data?.data ?? [],
		rowCount: data?.count ?? 0,
		enableFilters: true,
		enableHiding: true,
		enableFacetedValues: true,
		state: {
			isLoading,
			columnFilters,
			globalFilter,
			pagination,
			sorting,
			showAlertBanner: isError,
		},
		onColumnFiltersChange: setColumnFilters,
		onGlobalFilterChange: setGlobalFilter,
		onPaginationChange: setPagination,
		onSortingChange: setSorting,
		initialState: {
			...defaultOptions.initialState,
			sorting: [{ id: "ban_id", desc: true }],
			columnVisibility: {
				source_id: false,
			},
		},
	});

	return <SortableTable table={table} title={"Player IP History"} hideHeader />;
};
