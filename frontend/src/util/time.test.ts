import { format } from "date-fns/format";
import { describe, expect, test } from "vitest";
import { isEmptyDate, renderTimestamp } from "./time.ts";

describe("isEmptyDate", () => {
	test("treats epoch and pre-epoch dates as empty", () => {
		expect(isEmptyDate(new Date(0))).toBe(true);
		expect(isEmptyDate(new Date(-1000))).toBe(true);
		expect(isEmptyDate(new Date("1969-12-31T17:00:00"))).toBe(true);
	});

	test("treats real dates as non-empty", () => {
		expect(isEmptyDate(new Date())).toBe(false);
		expect(isEmptyDate(new Date("2024-01-01"))).toBe(false);
	});
});

describe("renderTimestamp", () => {
	test("renders nothing for missing timestamps", () => {
		expect(renderTimestamp(undefined)).toBe("");
		expect(renderTimestamp({ seconds: 0n, nanos: 0 })).toBe("");
	});

	test("renders real timestamps", () => {
		const expected = format(new Date(1704067200 * 1000), "yyyy-MM-dd HH:mm");
		expect(renderTimestamp({ seconds: 1704067200n, nanos: 0 })).toBe(expected);
	});
});
