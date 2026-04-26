import { intervalToDuration } from "date-fns";
import { z } from "zod/v4";
export enum RowsPerPage {
	Ten = 10,
	TwentyFive = 25,
	Fifty = 50,
	Hundred = 100,
}

export const isPermanentBan = (start: Date, end: Date): boolean => {
	const dur = intervalToDuration({
		start,
		end,
	});
	const { years } = dur;
	return years != null && years > 5;
};

export const commonTableSearchSchema = z.object({
	pageIndex: z.number().optional().catch(0),
	pageSize: z.number().optional().catch(RowsPerPage.TwentyFive),
	sortOrder: z.enum(["desc", "asc"]).optional().catch("desc"),
});

export type Order = "asc" | "desc";
