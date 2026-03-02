import Typography from "@mui/material/Typography";
import {
	createColumnHelper,
	getCoreRowModel,
	getPaginationRowModel,
	type OnChangeFn,
	type PaginationState,
	type TableOptions,
	useReactTable,
} from "@tanstack/react-table";
import { useMemo } from "react";
import type { PersonConnection } from "../../schema/people.ts";
import type { LazyResult } from "../../util/table.ts";
import { renderDateTime } from "../../util/time.ts";
import { DataTable } from "./DataTable.tsx";

const columnHelper = createColumnHelper<PersonConnection>();

export const IPHistoryTable = ({
	connections,
	isLoading,
	manualPaging = true,
	pagination,
	setPagination,
}: {
	connections: LazyResult<PersonConnection>;
	isLoading: boolean;
	manualPaging?: boolean;
	pagination?: PaginationState;
	setPagination?: OnChangeFn<PaginationState>;
}) => {
	const columns = useMemo(() => {
		return [
			columnHelper.accessor("created_on", {
				header: "Created",
				size: 120,
				cell: (info) => (
					<Typography>{renderDateTime(info.getValue())}</Typography>
				),
			}),
			columnHelper.accessor("persona_name", {
				header: "Name",
				cell: (info) => <Typography>{info.getValue()}</Typography>,
			}),
			columnHelper.accessor("ip_addr", {
				header: "IP Address",
				size: 120,
				cell: (info) => <Typography>{info.getValue()}</Typography>,
			}),
			columnHelper.accessor("server_id", {
				header: "Server",
				size: 120,
				cell: (info) => (
					<Typography>
						{connections.data[info.row.index].server_name_short}
					</Typography>
				),
			}),
		];
	}, [connections.data]);

	const opts: TableOptions<PersonConnection> = {
		data: connections.data,
		columns: columns,
		getCoreRowModel: getCoreRowModel(),
		manualPagination: manualPaging,
		autoResetPageIndex: true,
		...(manualPaging
			? {}
			: {
					manualPagination: false,
					onPaginationChange: setPagination,
					getPaginationRowModel: getPaginationRowModel(),
					state: { pagination },
				}),
	};

	const table = useReactTable(opts);

	return <DataTable table={table} isLoading={isLoading} />;
};
