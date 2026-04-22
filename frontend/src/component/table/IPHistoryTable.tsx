import Typography from "@mui/material/Typography";
import {
	createMRTColumnHelper,
	type MRT_ColumnFiltersState,
	type MRT_PaginationState,
	type MRT_SortingState,
	useMaterialReactTable,
} from "material-react-table";
import { useMemo, useState } from "react";
import { renderTimestamp } from "../../util/time.ts";
import { createDefaultTableOptions } from "./options.ts";
import { SortableTable } from "./SortableTable.tsx";
import type { PersonConnection } from "../../rpc/network/v1/network_pb.ts";
import { queryConnections } from "../../rpc/network/v1/network-NetworkService_connectquery.ts";
import { useQuery } from "@connectrpc/connect-query";

``;

const columnHelper = createMRTColumnHelper<PersonConnection>();
const defaultOptions = createDefaultTableOptions<PersonConnection>();

export const IPHistoryTable = ({ steamId }: { steamId: bigint }) => {
	const [columnFilters, setColumnFilters] = useState<MRT_ColumnFiltersState>([]);
	const [globalFilter, setGlobalFilter] = useState("");
	const [sorting, setSorting] = useState<MRT_SortingState>([]);
	const [pagination, setPagination] = useState<MRT_PaginationState>({
		pageIndex: 0,
		pageSize: 50,
	});

	const sort = sorting.find((sort) => sort);

	const { data, isLoading, isError } = useQuery(queryConnections, {
		steamId: steamId.toString(),
		filter: {
			limit: BigInt(pagination.pageSize),
			offset: BigInt(pagination.pageIndex * pagination.pageSize),
			orderBy: sort ? sort.id : "created_on",
			desc: sort ? sort.desc : false,
		},
	});

	const columns = useMemo(() => {
		return [
			columnHelper.accessor("createdOn", {
				header: "Created",
				size: 120,
				Cell: ({ cell }) => <Typography>{renderTimestamp(cell.getValue())}</Typography>,
			}),
			columnHelper.accessor("personaName", {
				header: "Name",
				Cell: ({ cell }) => <Typography>{cell.getValue()}</Typography>,
			}),
			columnHelper.accessor("ipAddr", {
				header: "IP Address",
				size: 120,
				Cell: ({ cell }) => <Typography>{cell.getValue()}</Typography>,
			}),
			columnHelper.accessor("serverId", {
				header: "Server",
				size: 120,
				Cell: ({ row }) => <Typography>{row.original.serverNameShort}</Typography>,
			}),
		];
	}, []);

	const table = useMaterialReactTable({
		...defaultOptions,
		columns,
		data: data?.connection ?? [],
		// rowCount: data?.count ?? 0,
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
