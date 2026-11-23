import { describe, expect, test } from "vitest";
import { futureDateSchema, optionalCIDRSchema, optionalDiscordIDSchema, steamID64Schema } from "./validation.ts";

describe("steamID64Schema", () => {
	test("accepts real SteamID64 values", () => {
		expect(steamID64Schema.safeParse("76561198084134025").success).toBe(true);
	});

	test("rejects empty, non-numeric and out-of-range values", () => {
		for (const value of ["", "abc", "123", "76561197960265728", "76561198084134025.0"]) {
			expect(steamID64Schema.safeParse(value).success, value).toBe(false);
		}
	});
});

describe("optionalCIDRSchema", () => {
	test("accepts empty, plain IPv4 and valid CIDR", () => {
		for (const value of ["", "1.2.3.4", "1.2.3.4/0", "10.0.0.0/8", "192.168.1.1/32"]) {
			expect(optionalCIDRSchema.safeParse(value).success, value).toBe(true);
		}
	});

	test("rejects malformed addresses and prefix lengths", () => {
		for (const value of ["1.2.3.4/33", "1.2.3.4/-1", "1.2.3.999", "1.2.3.4/1/2", "hostname/24"]) {
			expect(optionalCIDRSchema.safeParse(value).success, value).toBe(false);
		}
	});
});

describe("other helpers", () => {
	test("futureDateSchema rejects past dates", () => {
		expect(futureDateSchema.safeParse(new Date(Date.now() + 60_000)).success).toBe(true);
		expect(futureDateSchema.safeParse(new Date(Date.now() - 60_000)).success).toBe(false);
	});

	test("optionalDiscordIDSchema only accepts digits or empty", () => {
		expect(optionalDiscordIDSchema.safeParse("").success).toBe(true);
		expect(optionalDiscordIDSchema.safeParse("123456789").success).toBe(true);
		expect(optionalDiscordIDSchema.safeParse("abc").success).toBe(false);
	});
});
