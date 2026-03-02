import CheckIcon from "@mui/icons-material/Check";
import CloseIcon from "@mui/icons-material/Close";
import TableCell from "@mui/material/TableCell";

export const TableCellBool = ({ enabled }: { enabled: boolean }) => {
	return (
		<TableCell>
			{enabled ? (
				<CheckIcon color={"success"} />
			) : (
				<CloseIcon color={"error"} />
			)}
		</TableCell>
	);
};
