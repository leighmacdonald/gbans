import { expect, test } from "vitest";
import { Team } from "../../rpc/stats/v1/stats_pb.ts";
import { blu, red } from "../../theme.ts";
import { overviewUrls, parseWorldFile, playerColor, teamColorOf, worldToPixel } from "./killMap.ts";

// pl_upward_overview.pgw
const upwardPgw = "0.076200000000\n0.0\n0.0\n-0.076200000000\n-63.969900000000\n44.462700000000\n";

test("parseWorldFile parses a valid pgw", () => {
	const wf = parseWorldFile(upwardPgw);
	expect(wf).not.toBeNull();
	expect(wf?.pixelSizeX).toBeCloseTo(0.0762, 6);
	expect(wf?.pixelSizeY).toBeCloseTo(-0.0762, 6);
	expect(wf?.topLeftX).toBeCloseTo(-63.9699, 4);
	expect(wf?.topLeftY).toBeCloseTo(44.4627, 4);
});

test("parseWorldFile rejects malformed input", () => {
	expect(parseWorldFile("")).toBeNull();
	expect(parseWorldFile("1\n2\n3\n")).toBeNull();
	expect(parseWorldFile("a\nb\nc\nd\ne\nf\n")).toBeNull();
	expect(parseWorldFile("0\n0\n0\n0\n0\n0\n")).toBeNull();
});

test("worldToPixel maps the top-left world corner to pixel origin", () => {
	const wf = parseWorldFile(upwardPgw);
	expect(wf).not.toBeNull();
	if (!wf) {
		return;
	}
	const origin = worldToPixel(wf, wf.topLeftX, wf.topLeftY);
	expect(origin.px).toBeCloseTo(0, 6);
	expect(origin.py).toBeCloseTo(0, 6);
	// One pixel right/down in world units lands on pixel (1, 1).
	const next = worldToPixel(wf, wf.topLeftX + wf.pixelSizeX, wf.topLeftY + wf.pixelSizeY);
	expect(next.px).toBeCloseTo(1, 6);
	expect(next.py).toBeCloseTo(1, 6);
});

test("overviewUrls follows the <map>_overview naming", () => {
	expect(overviewUrls("pl_upward")).toEqual({
		image: "/maps/pl_upward_overview.png",
		worldFile: "/maps/pl_upward_overview.pgw",
	});
});

test("playerColor is deterministic and varies by steamId", () => {
	const a = playerColor("76561197960287930");
	const b = playerColor("76561197960287930");
	const c = playerColor("76561198066450438");
	expect(a).toBe(b);
	expect(a).not.toBe(c);
	expect(a).toMatch(/^hsl\(\d{1,3}, 75%, 50%\)$/);
});

test("teamColorOf maps teams to theme colors", () => {
	expect(teamColorOf(Team.BLU)).toBe(blu);
	expect(teamColorOf(Team.RED)).toBe(red);
});
