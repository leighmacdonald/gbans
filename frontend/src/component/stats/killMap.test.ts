import { expect, test } from "vitest";
import { Team } from "../../rpc/stats/v1/stats_pb.ts";
import { blu, red } from "../../theme.ts";
import {
	eventMarkerColor,
	eventTypeColor,
	eventTypeStyle,
	HAMMER_UNIT_IN_METERS,
	HIGHLIGHT_DEATH,
	HIGHLIGHT_KILL,
	overviewPixelBounds,
	overviewUrls,
	paddedOverviewPixelBounds,
	parseReferenceFile,
	parseWorldFile,
	pixelBoundsToLatLng,
	playerColor,
	REFERENCE_AMMO_COLOR,
	REFERENCE_HEALTH_COLOR,
	referencePointLabel,
	referencePointStyle,
	referenceUrls,
	teamColorOf,
	worldToPixel,
} from "./killMap.ts";

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

test("worldToPixel converts Hammer units to the metre-based world file", () => {
	const wf = parseWorldFile(upwardPgw);
	expect(wf).not.toBeNull();
	if (!wf) {
		return;
	}
	const origin = worldToPixel(wf, wf.topLeftX / HAMMER_UNIT_IN_METERS, wf.topLeftY / HAMMER_UNIT_IN_METERS);
	expect(origin.px).toBeCloseTo(0, 6);
	expect(origin.py).toBeCloseTo(0, 6);
	// One pixel right/down in Hammer units lands on pixel (1, 1).
	const next = worldToPixel(
		wf,
		(wf.topLeftX + wf.pixelSizeX) / HAMMER_UNIT_IN_METERS,
		(wf.topLeftY + wf.pixelSizeY) / HAMMER_UNIT_IN_METERS,
	);
	expect(next.px).toBeCloseTo(1, 6);
	expect(next.py).toBeCloseTo(1, 6);
});

test("worldToPixel places a Snakewater kill inside the overview", () => {
	const wf = parseWorldFile("0.076200000000\n0.0\n0.0\n-0.076200000000\n-113.499900000000\n73.037700000000\n");
	expect(wf).not.toBeNull();
	if (!wf) {
		return;
	}
	const point = worldToPixel(wf, 1487.75, 244.875);
	expect(point.px).toBeCloseTo(1985.42, 2);
	expect(point.py).toBeCloseTo(876.88, 2);
});

test("the overview viewport contains the image plus a buffer", () => {
	const size = { width: 3245, height: 1815 };
	const view = paddedOverviewPixelBounds(size);
	expect(view.minX).toBeLessThan(0);
	expect(view.minY).toBeLessThan(0);
	expect(view.maxX).toBeGreaterThan(size.width);
	expect(view.maxY).toBeGreaterThan(size.height);
	expect(pixelBoundsToLatLng(size.height, overviewPixelBounds(size))).toEqual([
		[size.height, 0],
		[0, size.width],
	]);
});

test("overviewUrls follows the <map>_overview naming", () => {
	expect(overviewUrls("pl_upward")).toEqual({
		image: "/maps/pl_upward_overview.png",
		worldFile: "/maps/pl_upward_overview.pgw",
	});
});

test("referenceUrls prefers the GeoJSON reference and falls back to JSON", () => {
	expect(referenceUrls("pl_upward")).toEqual({
		primary: "/maps/pl_upward_reference.geojson",
		fallback: "/maps/pl_upward_reference.json",
	});
});

test("parseReferenceFile keeps Hammer-unit reference points and ignores preview coordinates", () => {
	const points = parseReferenceFile({
		type: "FeatureCollection",
		features: [
			{
				type: "Feature",
				geometry: { type: "Point", coordinates: [0.0001, 0.00002] },
				properties: {
					classname: "team_control_point",
					targetname: "cp_3",
					team: "",
					hx: 527.999,
					hy: 207.999,
					hz: 67.7749,
				},
			},
			{
				type: "Feature",
				geometry: { type: "Point", coordinates: [0.0004, -0.0003] },
				properties: { classname: "info_player_teamspawn", targetname: "", team: "3", hx: 2120, hy: 376, hz: 0 },
			},
			{
				type: "Feature",
				geometry: { type: "LineString", coordinates: [[0, 0]] },
				properties: { classname: "ignored", hx: 1, hy: 2, hz: 3 },
			},
			{
				type: "Feature",
				geometry: { type: "Point", coordinates: [0, 0] },
				properties: { classname: "broken", hx: "nowhere", hy: 2, hz: 3 },
			},
		],
	});
	expect(points).toEqual([
		{ classname: "team_control_point", targetname: "cp_3", team: "", x: 527.999, y: 207.999, z: 67.7749 },
		{ classname: "info_player_teamspawn", targetname: "", team: "3", x: 2120, y: 376, z: 0 },
	]);
	expect(points?.[0] ? referencePointLabel(points[0]) : "").toBe("cp_3");
	expect(points?.[1] ? referencePointLabel(points[1]) : "").toBe("info_player_teamspawn · team 3");
	expect(points?.[0] ? referencePointStyle(points[0]).radius : 0).toBe(5);
	expect(parseReferenceFile({ type: "FeatureCollection" })).toBeNull();
});

test("ammopacks and health kits use distinct colors", () => {
	const ammo = referencePointStyle({
		classname: "item_ammopack_medium",
		targetname: "",
		team: "",
		x: 1546,
		y: -56,
		z: -14.7387,
	});
	const health = referencePointStyle({
		classname: "item_healthkit_medium",
		targetname: "",
		team: "",
		x: 1548,
		y: 0,
		z: -14.7639,
	});
	expect(ammo.color).toBe(REFERENCE_AMMO_COLOR);
	expect(health.color).toBe(REFERENCE_HEALTH_COLOR);
	expect(ammo.color).not.toBe(health.color);
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

test("selection highlights use dark green for kills and dark red for deaths", () => {
	expect(HIGHLIGHT_KILL).toBe("#1b5e20");
	expect(HIGHLIGHT_DEATH).toBe("#7f1d1d");
});

test("positional event types have distinct layer colors", () => {
	expect(eventTypeColor("kill")).toBe("#e53935");
	expect(eventTypeColor("building_built")).toBe("#00c853");
	expect(eventTypeColor("building_destroyed")).toBe("#ff6d00");
	expect(eventTypeColor("kill")).not.toBe(eventTypeColor("building_built"));
	// Unknown future types still get a deterministic color.
	expect(eventTypeColor("custom_type")).toBe(eventTypeColor("custom_type"));
	expect(eventTypeStyle("kill").radius).toBeLessThan(eventTypeStyle("building_built").radius);
});

test("building_built sub-layers have distinct colors", () => {
	expect(eventTypeColor("building_built_sentry")).toBe("#00c853");
	expect(eventTypeColor("building_built_dispenser")).toBe("#00b8d4");
	expect(eventTypeColor("building_built_teleporter")).toBe("#7c4dff");
	expect(eventTypeStyle("building_built_sentry").radius).toBe(5);
});

test("building markers use legend colors regardless of team", () => {
	for (const layer of [
		"building_built_sentry",
		"building_built_dispenser",
		"building_built_teleporter",
		"building_destroyed",
	]) {
		expect(eventMarkerColor(layer, Team.BLU)).toBe(eventTypeColor(layer));
		expect(eventMarkerColor(layer, Team.RED)).toBe(eventTypeColor(layer));
	}
	// Kills keep team colors so the attacking side stays visible.
	expect(eventMarkerColor("kill", Team.BLU)).toBe(blu);
	expect(eventMarkerColor("kill", Team.RED)).toBe(red);
	expect(eventMarkerColor("kill", Team.UNASSIGNED_UNSPECIFIED)).toBe(eventTypeColor("kill"));
});
