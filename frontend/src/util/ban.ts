import { addDays, addHours, addMinutes, addMonths, addWeeks, addYears } from "date-fns";

export type BanDuration =
	| "15m"
	| "6h"
	| "12h"
	| "1d"
	| "2d"
	| "3d"
	| "1w"
	| "2w"
	| "1mo"
	| "6mo"
	| "1y"
	| "permanent"
	| "custom";

type BanDurationOption = {
	value: BanDuration;
	label: string;
	getExpiration?: (now: Date) => Date;
};

export const banDurationOptions: BanDurationOption[] = [
	{ value: "15m", label: "15 minutes", getExpiration: (now) => addMinutes(now, 15) },
	{ value: "6h", label: "6 hours", getExpiration: (now) => addHours(now, 6) },
	{ value: "12h", label: "12 hours", getExpiration: (now) => addHours(now, 12) },
	{ value: "1d", label: "1 day", getExpiration: (now) => addDays(now, 1) },
	{ value: "2d", label: "2 days", getExpiration: (now) => addDays(now, 2) },
	{ value: "3d", label: "3 days", getExpiration: (now) => addDays(now, 3) },
	{ value: "1w", label: "1 week", getExpiration: (now) => addWeeks(now, 1) },
	{ value: "2w", label: "2 weeks", getExpiration: (now) => addWeeks(now, 2) },
	{ value: "1mo", label: "1 month", getExpiration: (now) => addMonths(now, 1) },
	{ value: "6mo", label: "6 months", getExpiration: (now) => addMonths(now, 6) },
	{ value: "1y", label: "1 year", getExpiration: (now) => addYears(now, 1) },
	{ value: "permanent", label: "Permanent", getExpiration: (now) => addYears(now, 10) },
	{ value: "custom", label: "Custom" },
];

export const banDurationValues = banDurationOptions.map((option) => option.value);

export const banExpirationFromDuration = (duration: BanDuration, now: Date = new Date()): Date | undefined => {
	return banDurationOptions.find((option) => option.value === duration)?.getExpiration?.(now);
};
