import Tooltip from "@mui/material/Tooltip";
import Typography from "@mui/material/Typography";
import { formatDistanceStrict } from "date-fns";
import { formatDistanceToNowStrict } from "date-fns/formatDistanceToNowStrict";
import React from "react";

interface DataTableRelativeDateFieldProps {
	date?: Date;
	compareDate?: Date;
	suffix?: boolean;
}

export const TableCellRelativeDateField = React.memo(
	({ date, compareDate, suffix = false }: DataTableRelativeDateFieldProps) => {
		if (!date) {
			return null;
		}
		const opts = {
			addSuffix: suffix,
		};
		return (
			<div>
				<Tooltip title={date.toUTCString()}>
					<Typography variant={"body1"}>
						{date.getFullYear() < 2000
							? ""
							: compareDate
								? formatDistanceStrict(date, compareDate, opts)
								: formatDistanceToNowStrict(date, opts)}
					</Typography>
				</Tooltip>
			</div>
		);
	},
	(prev, next) => prev.date === next.date && prev.compareDate === next.compareDate && prev.suffix === next.suffix,
);
