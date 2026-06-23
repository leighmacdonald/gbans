import CheckIcon from "@mui/icons-material/Check";
import CloseIcon from "@mui/icons-material/Close";
import React from "react";

/* For use with column defs that already output a <td> wrapper */
export const BoolCell = React.memo(({ enabled }: { enabled: boolean }) => {
	return enabled ? <CheckIcon color={"success"} /> : <CloseIcon color={"error"} />;
});
