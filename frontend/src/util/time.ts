import { type Timestamp, timestampDate } from "@bufbuild/protobuf/wkt";
import { formatDistance, parseISO, parseJSON } from "date-fns";
import { format } from "date-fns/format";
import { isAfter } from "date-fns/fp";

export const durationToMs = (d: number) => d / 1000;

export const durationToSec = (d: number) => d / 1000 / 1000;

/**
 * Converts a golang duration value to a string representation.
 * @param d
 */
export const durationString = (d: number) => {
	const secs = durationToSec(d);
	let hours: number | string = Math.floor(secs / 3600);
	let minutes: number | string = Math.floor((secs - hours * 3600) / 60);
	let seconds: number | string = secs - hours * 3600 - minutes * 60;

	if (hours < 10) {
		hours = `0${hours}`;
	}

	if (minutes < 10) {
		minutes = `0${minutes}`;
	}

	if (seconds < 10) {
		seconds = `0${seconds}`;
	}

	return `${hours}:${minutes}:${seconds}`;
};

export const parseDateTime = (t: string | Date): Date => {
	if (t instanceof Date) {
		return t;
	}

	return parseISO(t);
};

export const renderTimestamp = (ts?: Timestamp): string => {
	if (!ts) return "";

	return renderDateTime(timestampDate(ts));
};

export const renderDateTime = (t: Date): string => {
	return format(t, "yyyy-MM-dd HH:mm");
};

export const renderDate = (t: Date): string => {
	return format(t, "yyyy-MM-dd");
};

export const renderTime = (t: Date): string => {
	return format(t, "HH:mm");
};

export const isValidSteamDate = (date: Date) => isAfter(new Date(2000, 0, 0), date);

export const renderTimeDistance = (t1: Date | string, t2?: Date | string): string => {
	if (typeof t1 === "string") {
		t1 = parseJSON(t1);
	}
	if (!t2) {
		t2 = new Date();
	}
	if (typeof t2 === "string") {
		t2 = parseJSON(t2);
	}
	return formatDistance(t1, t2, {
		addSuffix: true,
	});
};
