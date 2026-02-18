import { expect, test } from "vitest";
import { sum } from "./lists.ts";

export const noop = (): void => {};

test("sum", () => {
	expect(sum([1, 2, 3])).toBe(6);
});
