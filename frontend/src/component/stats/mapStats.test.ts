import { create } from "@bufbuild/protobuf";
import { TimestampSchema } from "@bufbuild/protobuf/wkt";
import { format } from "date-fns";
import { expect, test } from "vitest";

// The chart buckets are calendar based; pin the timezone so the expected
// labels are deterministic regardless of where the suite runs.
process.env.TZ = "UTC";

import {
	MapPrefixStatSchema,
	MapStatSchema,
	MapStatsPointSchema,
	MapStatsSeriesSchema,
	TimeBucket,
} from "../../rpc/stats/v1/stats_pb.ts";
import {
	buildMapStatsChart,
	buildMapStatsPrefixChart,
	buildMapStatsTop,
	formatMetricValue,
	formatPlaytimeHours,
	mapStatsMetricLabel,
	mapStatsRangeLabel,
	mapStatsWindow,
} from "./mapStats.ts";

const ts = (iso: string) =>
	create(TimestampSchema, {
		seconds: BigInt(Math.floor(Date.parse(iso) / 1000)),
		nanos: 0,
	});

const point = (iso: string, values: { durationMs?: string; matchesPlayed?: string; matchesDecided?: string } = {}) =>
	create(MapStatsPointSchema, { dateBucket: ts(iso), ...values });

const series = (mapId: number, mapName: string, points: ReturnType<typeof point>[]) =>
	create(MapStatsSeriesSchema, { mapId, mapName, points });

test("mapStatsWindow resolves rolling ranges", () => {
	const now = new Date(2026, 3, 15, 10, 30, 0);

	const w7 = mapStatsWindow("7d", now);
	expect(w7.group).toBe(TimeBucket.DAILY);
	expect(format(w7.start!, "yyyy-MM-dd")).toBe("2026-04-09");
	expect(format(w7.end!, "yyyy-MM-dd HH:mm:ss")).toBe("2026-04-15 23:59:59");

	const w90 = mapStatsWindow("90d", now);
	expect(w90.group).toBe(TimeBucket.DAILY);
	expect(format(w90.start!, "yyyy-MM-dd")).toBe("2026-01-16");

	const w1y = mapStatsWindow("1y", now);
	expect(w1y.group).toBe(TimeBucket.WEEKLY);
	expect(format(w1y.start!, "yyyy-MM-dd")).toBe("2025-04-15");

	const wAll = mapStatsWindow("all", now);
	expect(wAll.group).toBe(TimeBucket.MONTHLY);
	expect(wAll.start).toBeUndefined();
	expect(wAll.end).toBeUndefined();
});

test("buildMapStatsChart fills gaps with zeros for daily buckets", () => {
	const data = buildMapStatsChart(
		[
			series(1, "cp_a", [
				point("2026-01-01T00:00:00Z", { durationMs: "3600000" }),
				point("2026-01-04T00:00:00Z", { durationMs: "7200000" }),
			]),
			// A series with only zero values is dropped from the chart.
			series(2, "cp_zero", [point("2026-01-01T00:00:00Z", { durationMs: "0" })]),
		],
		"duration",
		TimeBucket.DAILY,
		new Date("2026-01-01T00:00:00Z"),
		new Date("2026-01-05T00:00:00Z"),
	);

	expect(data.labels).toEqual(["Jan 1", "Jan 2", "Jan 3", "Jan 4", "Jan 5"]);
	expect(data.series).toHaveLength(1);
	expect(data.series[0].label).toBe("cp_a");
	expect(data.series[0].data).toEqual([1, 0, 0, 2, 0]);
});

test("buildMapStatsChart merges days into weekly buckets", () => {
	const data = buildMapStatsChart(
		[
			series(1, "cp_a", [
				point("2026-01-01T00:00:00Z", { durationMs: "3600000" }),
				point("2026-01-03T00:00:00Z", { durationMs: "7200000" }),
				point("2026-01-06T00:00:00Z", { durationMs: "3600000" }),
			]),
		],
		"duration",
		TimeBucket.WEEKLY,
		new Date("2026-01-01T00:00:00Z"),
		new Date("2026-01-08T00:00:00Z"),
	);

	// Jan 1 2026 is a Thursday: the window spans the weeks starting Dec 29 and Jan 5.
	expect(data.labels).toEqual(["Dec 29", "Jan 5"]);
	expect(data.series[0].data).toEqual([3, 1]);
});

test("buildMapStatsChart derives the window from the points when no range is given", () => {
	const data = buildMapStatsChart(
		[
			series(1, "cp_a", [
				point("2026-01-01T00:00:00Z", { matchesPlayed: "2" }),
				point("2026-01-03T00:00:00Z", { matchesPlayed: "1" }),
			]),
		],
		"matches",
		TimeBucket.DAILY,
	);

	expect(data.labels).toEqual(["Jan 1", "Jan 2", "Jan 3"]);
	expect(data.series[0].data).toEqual([2, 0, 1]);
});

test("buildMapStatsChart returns empty data for no series", () => {
	const data = buildMapStatsChart(
		[],
		"duration",
		TimeBucket.DAILY,
		new Date("2026-01-01T00:00:00Z"),
		new Date("2026-01-02T00:00:00Z"),
	);
	expect(data.labels).toEqual([]);
	expect(data.series).toEqual([]);
});

test("buildMapStatsTop converts playtime to hours and drops zero maps", () => {
	const top = buildMapStatsTop([
		create(MapStatSchema, { mapId: 1, mapName: "cp_badlands", totalDurationMs: "7200000" }),
		create(MapStatSchema, { mapId: 2, mapName: "cp_process", totalDurationMs: "3600000" }),
		create(MapStatSchema, { mapId: 3, mapName: "cp_zero", totalDurationMs: "0" }),
	]);

	expect(top).toEqual([
		{ label: "cp_badlands", value: 2 },
		{ label: "cp_process", value: 1 },
	]);
});

test("buildMapStatsPrefixChart converts playtime to hours, sorts and drops zero entries", () => {
	const data = buildMapStatsPrefixChart(
		[
			create(MapPrefixStatSchema, {
				prefix: "cp_",
				mapsPlayed: "3",
				matchesPlayed: "4",
				matchesDecided: "2",
				totalDurationMs: "18000000",
			}),
			create(MapPrefixStatSchema, {
				prefix: "pl_",
				mapsPlayed: "1",
				matchesPlayed: "1",
				matchesDecided: "1",
				totalDurationMs: "2700000",
			}),
			create(MapPrefixStatSchema, {
				prefix: "koth_",
				mapsPlayed: "1",
				matchesPlayed: "1",
				matchesDecided: "1",
				totalDurationMs: "3600000",
			}),
			create(MapPrefixStatSchema, {
				prefix: "other",
				mapsPlayed: "1",
				matchesPlayed: "1",
				matchesDecided: "0",
				totalDurationMs: "0",
			}),
		],
		"duration",
	);

	expect(data).toEqual([
		{ label: "cp_", value: 5, matches: 4, maps: 3 },
		{ label: "koth_", value: 1, matches: 1, maps: 1 },
		{ label: "pl_", value: 0.75, matches: 1, maps: 1 },
	]);
});

test("buildMapStatsPrefixChart follows the selected metric and breaks ties alphabetically", () => {
	const data = buildMapStatsPrefixChart(
		[
			create(MapPrefixStatSchema, {
				prefix: "pl_",
				mapsPlayed: "1",
				matchesPlayed: "1",
				matchesDecided: "1",
				totalDurationMs: "2700000",
			}),
			create(MapPrefixStatSchema, {
				prefix: "cp_",
				mapsPlayed: "3",
				matchesPlayed: "4",
				matchesDecided: "2",
				totalDurationMs: "18000000",
			}),
			create(MapPrefixStatSchema, {
				prefix: "other",
				mapsPlayed: "1",
				matchesPlayed: "1",
				matchesDecided: "0",
				totalDurationMs: "900000",
			}),
		],
		"matches",
	);

	// Under "matches" the pl_/other tie (both 1) sorts alphabetically.
	expect(data.map((entry) => entry.label)).toEqual(["cp_", "other", "pl_"]);
	expect(data[0].value).toBe(4);

	const decided = buildMapStatsPrefixChart(
		[
			create(MapPrefixStatSchema, {
				prefix: "pl_",
				mapsPlayed: "1",
				matchesPlayed: "1",
				matchesDecided: "1",
				totalDurationMs: "2700000",
			}),
			create(MapPrefixStatSchema, {
				prefix: "cp_",
				mapsPlayed: "3",
				matchesPlayed: "4",
				matchesDecided: "2",
				totalDurationMs: "18000000",
			}),
			create(MapPrefixStatSchema, {
				prefix: "other",
				mapsPlayed: "1",
				matchesPlayed: "1",
				matchesDecided: "0",
				totalDurationMs: "900000",
			}),
		],
		"decided",
	);

	// other has no decided matches, so it is dropped under the "decided" metric.
	expect(decided.map((entry) => entry.label)).toEqual(["cp_", "pl_"]);
});

test("formatPlaytimeHours renders human playtime", () => {
	expect(formatPlaytimeHours("12600000")).toBe("3.5h");
	expect(formatPlaytimeHours("0")).toBe("0.0h");
	expect(formatPlaytimeHours("12345600000")).toBe("3,429h");
	expect(formatPlaytimeHours("not-a-number")).toBe("0h");
});

test("formatMetricValue formats per metric", () => {
	expect(formatMetricValue(2.5, "duration")).toBe("2.5h");
	expect(formatMetricValue(1234, "matches")).toBe("1,234");
	expect(formatMetricValue(5, "decided")).toBe("5");
});

test("range and metric labels resolve", () => {
	expect(mapStatsRangeLabel("90d")).toBe("Last 90 days");
	expect(mapStatsMetricLabel("duration")).toBe("Playtime");
});
