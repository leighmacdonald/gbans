import Grid from "@mui/material/Grid";
import TableCell from "@mui/material/TableCell";
import Typography from "@mui/material/Typography";
import { useQuery } from "@tanstack/react-query";
import { createFileRoute } from "@tanstack/react-router";
import {
	createMRTColumnHelper,
	type MRT_ColumnFiltersState,
	type MRT_PaginationState,
	type MRT_SortingState,
	useMaterialReactTable,
} from "material-react-table";
import { useMemo, useState } from "react";
import { apiGetConnections } from "../api";
import { TextLink } from "../component/TextLink.tsx";
import { createDefaultTableOptions } from "../component/table/options.ts";
import { SortableTable } from "../component/table/SortableTable.tsx";
import type { PersonConnection } from "../schema/people.ts";
import { renderDateTime } from "../util/time.ts";

const columnHelper = createMRTColumnHelper<PersonConnection>();
const defaultOptions = createDefaultTableOptions<PersonConnection>();

export const Route = createFileRoute("/_mod/admin/network/playersbyip")({
	component: AdminNetworkPlayersByCIDR,
	head: ({ match }) => ({
		meta: [{ name: "description", content: "Find players by IP address" }, match.context.title("Players By IP")],
	}),
});

function AdminNetworkPlayersByCIDR() {
	const [columnFilters, setColumnFilters] = useState<MRT_ColumnFiltersState>([]);
	const [globalFilter, setGlobalFilter] = useState("");
	const [sorting, setSorting] = useState<MRT_SortingState>([]);
	const [pagination, setPagination] = useState<MRT_PaginationState>({
		pageIndex: 0,
		pageSize: 50,
	});

	const { data, isLoading, isError, isRefetching } = useQuery({
		queryKey: ["playersByIP", { columnFilters, globalFilter, pagination, sorting }],
		queryFn: async () => {
			const cidr = String(columnFilters.find((filter) => filter.id === "cidr")?.value ?? "");
			const steam_id = String(columnFilters.find((filter) => filter.id === "steam_id")?.value ?? "");
			const sort = sorting.find((sort) => sort);

			return await apiGetConnections({
				desc: sort ? sort.desc : true,
				limit: pagination.pageSize,
				offset: pagination.pageIndex * pagination.pageSize,
				order_by: sort ? sort.id : "created_on",
				source_id: steam_id,
				cidr: cidr ?? "",
			});
		},
	});

	const columns = useMemo(
		() => [
			columnHelper.accessor("server_id", {
				header: "Server",
				grow: false,
				//filterVariant: "multi-select",
				Cell: ({ row }) => (
					<TableCell>
						<Typography>{row.original.server_name_short}</Typography>
					</TableCell>
				),
			}),
			columnHelper.accessor("created_on", {
				grow: false,
				filterVariant: "date-range",
				header: "Created",
				Cell: ({ cell }) => <Typography>{renderDateTime(cell.getValue())}</Typography>,
			}),
			columnHelper.accessor("persona_name", {
				header: "Name",
				grow: true,
				enableSorting: false,
				Cell: ({ cell }) => (
					<TableCell>
						<Typography>{cell.getValue()}</Typography>
					</TableCell>
				),
			}),
			columnHelper.accessor("steam_id", {
				grow: false,
				header: "Steam ID",
				enableSorting: false,
				Cell: ({ cell }) => (
					<TableCell>
						<TextLink to={"/profile/$steamId"} params={{ steamId: cell.getValue() }}>
							{cell.getValue()}
						</TextLink>
					</TableCell>
				),
			}),
			columnHelper.accessor("ip_addr", {
				grow: false,
				header: "IP Address",
				Cell: ({ cell }) => (
					<TableCell>
						<Typography>{cell.getValue()}</Typography>
					</TableCell>
				),
			}),
		],
		[],
	);

	const table = useMaterialReactTable({
		...defaultOptions,
		columns,
		data: data?.data ?? [],
		rowCount: data?.count ?? 0,
		enableFilters: true,
		enableHiding: true,
		enableFacetedValues: true,
		state: {
			columnFilters,
			globalFilter,
			isLoading,
			pagination,
			showAlertBanner: isError,
			showProgressBars: isRefetching,
			sorting,
		},
		initialState: {
			...defaultOptions.initialState,
			sorting: [{ id: "ban_id", desc: true }],
			columnVisibility: {
				cidr_block_whitelist_id: false,
				address: true,
				created_on: true,
				updated_on: false,
			},
		},
		manualFiltering: true,
		manualPagination: true,
		manualSorting: true,
		onColumnFiltersChange: setColumnFilters,
		onGlobalFilterChange: setGlobalFilter,
		onPaginationChange: setPagination,
		onSortingChange: setSorting,
		enableRowActions: false,
	});

	return (
		<Grid container spacing={2}>
			<Grid size={{ xs: 12 }}>
				<SortableTable table={table} title={"Players By CIDR/IP"} />
			</Grid>
		</Grid>
	);
}
