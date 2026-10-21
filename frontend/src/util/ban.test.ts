import { addMinutes, addWeeks, addYears } from "date-fns";
import { describe, expect, test } from "vitest";
import { banDurationOptions, banDurationValues, banExpirationFromDuration } from "./ban.ts";

describe("banExpirationFromDuration", () => {
	const now = new Date("2026-01-01T00:00:00.000Z");

	test("offsets fixed durations from now", () => {
		expect(banExpirationFromDuration("15m", now)).toEqual(addMinutes(now, 15));
		expect(banExpirationFromDuration("2w", now)).toEqual(addWeeks(now, 2));
		expect(banExpirationFromDuration("permanent", now)).toEqual(addYears(now, 10));
	});

	test("returns undefined for custom durations", () => {
		expect(banExpirationFromDuration("custom", now)).toBeUndefined();
	});

	test("includes every option in the select values", () => {
		expect(banDurationValues).toEqual(banDurationOptions.map((option) => option.value));
	});
});
