import CheckIcon from "@mui/icons-material/Check";
import CloseIcon from "@mui/icons-material/Close";
import TableCell from "@mui/material/TableCell";
import React from "react";

export const TableCellBool = React.memo(({ enabled }: { enabled: boolean }) => {
	return <TableCell>{enabled ? <CheckIcon color={"success"} /> : <CloseIcon color={"error"} />}</TableCell>;
});
