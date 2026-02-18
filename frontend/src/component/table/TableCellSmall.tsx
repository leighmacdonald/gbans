import TableCell, { type TableCellProps } from "@mui/material/TableCell";

export const TableCellSmall = (props: TableCellProps) => {
	return (
		<TableCell {...props} padding={"none"}>
			{props.children}
		</TableCell>
	);
};
