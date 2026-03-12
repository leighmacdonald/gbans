import type { MRT_Row, MRT_RowData } from "material-react-table";
import { BanReason } from "../../schema/bans";

export const filterReason = <TData extends MRT_RowData>(
	row: MRT_Row<TData>,
	_: string,
	filterValue: TData["reason"],
): boolean => {
	return filterValue.length === 0 || filterValue.includes(BanReason.Any) || filterValue.includes(row.original.reason);
};
