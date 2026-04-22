import Tooltip from "@mui/material/Tooltip";
import Typography from "@mui/material/Typography";
import { formatDistanceStrict } from "date-fns";
import { formatDistanceToNowStrict } from "date-fns/formatDistanceToNowStrict";

interface DataTableRelativeDateFieldProps {
	date?: Date;
	compareDate?: Date;
	suffix?: boolean;
}

export const TableCellRelativeDateField = ({ date, compareDate, suffix = false }: DataTableRelativeDateFieldProps) => {
	if (!date) {
		return <></>;
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
};
