import { expect, test } from "vitest";
import { humanCount, steamConnectURL, steamSTVConnectURL } from "./strings.ts";

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

test("steamConnectURL uses the run launch pattern", () => {
	expect(steamConnectURL("108.181.3.247", 27015)).toBe("steam://run/440//+connect%20108.181.3.247:27015");
});

test("steamSTVConnectURL omits the password when none is set", () => {
	expect(steamSTVConnectURL("108.181.62.21", 27016)).toBe("steam://connect/108.181.62.21:27016");
	expect(steamSTVConnectURL("108.181.62.21", 27016, "")).toBe("steam://connect/108.181.62.21:27016");
});

test("steamSTVConnectURL urlencodes the stv password", () => {
	expect(steamSTVConnectURL("108.181.62.21", 27016, "password")).toBe("steam://connect/108.181.62.21:27016/password");
	expect(steamSTVConnectURL("108.181.62.21", 27016, "pass word&special#1")).toBe(
		"steam://connect/108.181.62.21:27016/pass%20word%26special%231",
	);
});
