import { expect, test } from "vitest";
import { sum } from "./lists.ts";

test("sum", () => {
	expect(sum([1, 2, 3])).toBe(6);
});
