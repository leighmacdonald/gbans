/** biome-ignore-all lint/correctness/noChildrenProp: form needs it */

import Grid from "@mui/material/Grid";
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
import { apiVotesQuery } from "../api/votes.ts";
import { PersonCell } from "../component/PersonCell.tsx";
import { BoolCell } from "../component/table/BoolCell.tsx";
import { createDefaultTableOptions } from "../component/table/options.ts";
import { SortableTable } from "../component/table/SortableTable.tsx";
import type { VoteResult } from "../schema/votes.ts";
import { renderDateTime } from "../util/time.ts";

export const Route = createFileRoute("/_mod/admin/votes")({
	component: AdminVotes,
	head: ({ match }) => ({
		meta: [{ name: "description", content: "Votes" }, match.context.title("Votes")],
	}),
});

const columnHelper = createMRTColumnHelper<VoteResult>();
const defaultOptions = createDefaultTableOptions<VoteResult>();

function AdminVotes() {
	const [columnFilters, setColumnFilters] = useState<MRT_ColumnFiltersState>([]);
	const [globalFilter, setGlobalFilter] = useState("");
	const [sorting, setSorting] = useState<MRT_SortingState>([]);
	const [pagination, setPagination] = useState<MRT_PaginationState>({
		pageIndex: 0,
		pageSize: 50,
	});

	const { data, isLoading, isError, isRefetching } = useQuery({
		queryKey: ["votes", {}],
		queryFn: async () => {
			const sort = sorting.find((sort) => sort);
			const source_id = String(columnFilters.find((filter) => filter.id === "source_id")?.value ?? "");
			const target_id = String(columnFilters.find((filter) => filter.id === "target_id")?.value ?? "");
			const success = sorting.find((success) => success);
			console.log(success);
			return apiVotesQuery({
				limit: pagination.pageSize,
				offset: pagination.pageIndex * pagination.pageSize,
				order_by: sort ? sort.id : "vote_id",
				desc: sort ? sort.desc : false,
				source_id: source_id ?? "",
				target_id: target_id ?? "",
				success: -1,
			});
		},
	});

	const columns = useMemo(
		() => [
			columnHelper.accessor("source_id", {
				header: "Initiator",
				grow: true,
				enableSorting: false,
				Cell: ({ row }) => (
					<PersonCell
						showCopy={true}
						steam_id={row.original.source_id}
						personaname={row.original.source_name}
						avatar_hash={row.original.source_avatar_hash}
					/>
				),
			}),
			columnHelper.accessor("target_id", {
				header: "Subject",
				grow: true,
				enableSorting: false,
				Cell: ({ row }) => {
					return (
						<PersonCell
							showCopy={true}
							steam_id={row.original.target_id}
							personaname={row.original.target_name}
							avatar_hash={row.original.target_avatar_hash}
						/>
					);
				},
			}),
			columnHelper.accessor("success", {
				header: "Passed",
				grow: false,
				enableSorting: false,
				filterVariant: "checkbox",
				Cell: ({ cell }) => {
					return <BoolCell enabled={cell.getValue()} />;
				},
			}),
			columnHelper.accessor("server_name", {
				header: "Server",
				filterVariant: "multi-select",
				enableSorting: false,
				grow: false,
				Cell: ({ cell }) => <Typography>{cell.getValue()}</Typography>,
			}),
			columnHelper.accessor("created_on", {
				header: "Created",
				grow: false,
				enableColumnFilter: false,
				Cell: ({ cell }) => <Typography>{renderDateTime(cell.getValue())}</Typography>,
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
		state: {
			columnFilters,
			globalFilter,
			isLoading,
			pagination,
			showAlertBanner: isError,
			showProgressBars: isRefetching,
			sorting,
		},
		manualFiltering: true,
		manualPagination: true,
		manualSorting: true,
		onColumnFiltersChange: setColumnFilters,
		onGlobalFilterChange: setGlobalFilter,
		onPaginationChange: setPagination,
		onSortingChange: setSorting,
		initialState: {
			...defaultOptions.initialState,
			sorting: [{ id: "updated_on", desc: true }],
			columnVisibility: {
				source_id: true,
				target_id: true,
				passed: true,
				server_name: true,
			},
		},
	});

	return (
		<Grid container spacing={2}>
			<Grid size={{ xs: 12 }}>
				<SortableTable table={table} title={"Vote History"} />
			</Grid>
		</Grid>
	);
}
