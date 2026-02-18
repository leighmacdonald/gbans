import { TablePagination } from "@mui/material";
import type { RowsPerPage } from "../../util/table.ts";

/**
 * A paginator for use when all data is available in the client, ie. no LazyTable
 *
 * @param count
 * @param page
 * @param rows
 * @param onRowsChange
 * @param onPageChange
 * @constructor
 */
export const PaginatorLocal = ({
	count,
	page,
	rows,
	onRowsChange,
	onPageChange,
}: {
	count: number;
	page: number;
	rows: number;
	onRowsChange: (rows: RowsPerPage) => void;
	onPageChange: (page: number) => void;
}) => {
	return (
		<TablePagination
			showFirstButton
			showLastButton
			padding={"none"}
			component={"div"}
			count={count}
			page={page}
			rowsPerPage={rows}
			onRowsPerPageChange={async (event) => {
				onRowsChange(Number(event.target.value));
			}}
			onPageChange={async (_, newPage: number) => {
				onPageChange(newPage);
			}}
		/>
	);
};
