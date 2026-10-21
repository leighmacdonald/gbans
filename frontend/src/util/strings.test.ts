import { expect, test } from "vitest";
import { humanCount } from "./strings.ts";

test("humanCount", () => {
	expect(humanCount(0)).toBe("0");
	expect(humanCount(999)).toBe("999");
	expect(humanCount(1000)).toBe("1k");
	expect(humanCount(1100)).toBe("1.1k");
	expect(humanCount(12345)).toBe("12.3k");
	expect(humanCount(1_000_000)).toBe("1m");
	expect(humanCount(-1100)).toBe("-1.1k");
	expect(humanCount(Number.NaN)).toBe("0");
});
