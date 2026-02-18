import Typography from "@mui/material/Typography";
import { type PropsWithChildren, useCallback, useState } from "react";

export const TableCellStringHidden = ({ children }: PropsWithChildren) => {
	const [hidden, setHidden] = useState(true);

	const onClick = useCallback(() => {
		setHidden((prev) => !prev);
	}, []);

	return (
		<Typography
			padding={"none"}
			onClick={onClick}
			sx={{
				"&": { textDecoration: "underline" },
				"&:hover": { cursor: "pointer" },
			}}
		>
			{hidden ? "Hidden" : children}
		</Typography>
	);
};
