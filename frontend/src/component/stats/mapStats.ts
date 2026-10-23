import { type Timestamp, timestampDate } from "@bufbuild/protobuf/wkt";
import {
	eachDayOfInterval,
	eachMonthOfInterval,
	eachWeekOfInterval,
	endOfDay,
	format,
	startOfDay,
	startOfMonth,
	startOfWeek,
	subDays,
	subYears,
} from "date-fns";
import type { MapStat, MapStatsSeries } from "../../rpc/stats/v1/stats_pb.ts";
import { TimeBucket } from "../../rpc/stats/v1/stats_pb.ts";

export type MapStatsRange = "7d" | "30d" | "90d" | "1y" | "all";

export const mapStatsRangeOptions: Array<{ value: MapStatsRange; label: string }> = [
	{ value: "7d", label: "Last 7 days" },
	{ value: "30d", label: "Last 30 days" },
	{ value: "90d", label: "Last 90 days" },
	{ value: "1y", label: "Last year" },
	{ value: "all", label: "All time" },
];

export const defaultMapStatsRange: MapStatsRange = "90d";

export type MapStatsMetric = "duration" | "matches" | "decided";

export const mapStatsMetricOptions: Array<{ value: MapStatsMetric; label: string }> = [
	{ value: "duration", label: "Playtime" },
	{ value: "matches", label: "Matches" },
	{ value: "decided", label: "Decided" },
];

export interface MapStatsWindow {
	start?: Date;
	end?: Date;
	group: TimeBucket;
}

/**
 * Resolves a rolling range into an explicit start/end window and the time
 * bucket used to group the chart series. The backend receives explicit
 * timestamps so the query is deterministic for the rendered page.
 */
export const mapStatsWindow = (range: MapStatsRange, now: Date = new Date()): MapStatsWindow => {
	switch (range) {
		case "7d":
			return { start: startOfDay(subDays(now, 6)), end: endOfDay(now), group: TimeBucket.DAILY };
		case "30d":
			return { start: startOfDay(subDays(now, 29)), end: endOfDay(now), group: TimeBucket.DAILY };
		case "90d":
			return { start: startOfDay(subDays(now, 89)), end: endOfDay(now), group: TimeBucket.DAILY };
		case "1y":
			return { start: startOfDay(subYears(now, 1)), end: endOfDay(now), group: TimeBucket.WEEKLY };
		case "all":
			return { group: TimeBucket.MONTHLY };
	}
};

export const mapStatsRangeLabel = (range: MapStatsRange): string =>
	mapStatsRangeOptions.find((option) => option.value === range)?.label ?? range;

export const mapStatsMetricLabel = (metric: MapStatsMetric): string =>
	mapStatsMetricOptions.find((option) => option.value === metric)?.label ?? metric;

const msToHours = (ms: number | string): number => Number(ms) / 3_600_000;

export const formatPlaytimeHours = (ms: number | string): string => {
	const hours = msToHours(ms);
	if (!Number.isFinite(hours)) {
		return "0h";
	}
	if (hours < 10) {
		return `${hours.toFixed(1)}h`;
	}

	return `${Math.round(hours).toLocaleString("en")}h`;
};

export const formatMetricValue = (value: number, metric: MapStatsMetric): string => {
	switch (metric) {
		case "duration":
			return `${value.toFixed(1)}h`;
		case "matches":
		case "decided":
			return value.toLocaleString("en");
	}
};

const metricValue = (point: MapStatsSeries["points"][number], metric: MapStatsMetric): number => {
	switch (metric) {
		case "duration":
			return msToHours(point.durationMs);
		case "matches":
			return Number(point.matchesPlayed);
		case "decided":
			return Number(point.matchesDecided);
	}
};

const normalizeBucket = (date: Date, group: TimeBucket): Date => {
	switch (group) {
		case TimeBucket.WEEKLY:
			return startOfWeek(date, { weekStartsOn: 1 });
		case TimeBucket.MONTHLY:
			return startOfMonth(date);
		default:
			return startOfDay(date);
	}
};

const bucketFormat = (group: TimeBucket): string => (group === TimeBucket.MONTHLY ? "MMM yyyy" : "MMM d");

export interface MapStatsChartData {
	labels: string[];
	series: Array<{ mapId: number; label: string; data: number[] }>;
}

/**
 * Converts RPC map stats series into chart data with one entry per bucket in
 * the window. Missing buckets are filled with zero so gaps do not compress
 * the x-axis.
 */
export const buildMapStatsChart = (
	series: MapStatsSeries[],
	metric: MapStatsMetric,
	group: TimeBucket,
	start?: Date,
	end?: Date,
): MapStatsChartData => {
	if (series.length === 0) {
		return { labels: [], series: [] };
	}

	const pointDates = series.flatMap((s) => s.points).map((point) => timestampDate(point.dateBucket as Timestamp));
	const rawStart =
		start ?? (pointDates.length > 0 ? new Date(Math.min(...pointDates.map((d) => d.getTime()))) : undefined);
	const rawEnd =
		end ?? (pointDates.length > 0 ? new Date(Math.max(...pointDates.map((d) => d.getTime()))) : undefined);

	if (!rawStart || !rawEnd) {
		return { labels: [], series: [] };
	}

	// Snap the bounds to bucket starts so the generated buckets and the
	// point lookup below share the same key space in every timezone.
	const interval = { start: normalizeBucket(rawStart, group), end: normalizeBucket(rawEnd, group) };
	const dates =
		group === TimeBucket.WEEKLY
			? eachWeekOfInterval(interval, { weekStartsOn: 1 })
			: group === TimeBucket.MONTHLY
				? eachMonthOfInterval(interval)
				: eachDayOfInterval(interval);

	const labels = dates.map((date) => format(normalizeBucket(date, group), bucketFormat(group)));
	const indexByBucket = new Map(dates.map((date, idx) => [normalizeBucket(date, group).toISOString(), idx]));

	const chartSeries = series
		.map((s) => {
			const values = new Array<number>(labels.length).fill(0);
			for (const point of s.points) {
				const idx = indexByBucket.get(
					normalizeBucket(timestampDate(point.dateBucket as Timestamp), group).toISOString(),
				);
				if (idx !== undefined) {
					values[idx] += metricValue(point, metric);
				}
			}

			return { mapId: s.mapId, label: s.mapName, data: values };
		})
		.filter((s) => s.data.some((value) => value > 0));

	return { labels, series: chartSeries };
};

/**
 * Converts the top maps from the table response into horizontal bar chart
 * data. The backend already orders them by total playtime.
 */
export const buildMapStatsTop = (topMaps: MapStat[]): Array<{ label: string; value: number }> =>
	topMaps
		.map((map) => ({ label: map.mapName, value: msToHours(map.totalDurationMs) }))
		.filter((entry) => entry.value > 0);
