import { Box, Typography } from "@mui/material";
import { BarChart, LineChart } from "@mui/x-charts";
import type { MapPrefixStat, MapStat } from "../../rpc/stats/v1/stats_pb.ts";
import {
	buildMapStatsPrefixChart,
	buildMapStatsTop,
	formatMetricValue,
	type MapStatsChartData,
	type MapStatsMetric,
	mapStatsMetricLabel,
} from "./mapStats.ts";

const chartHeight = 380;

// EmptyChart renders a centered placeholder so the paper keeps its height
// while there is no activity to plot.
const EmptyChart = ({ height }: { height: number }) => (
	<Box sx={{ height, display: "flex", alignItems: "center", justifyContent: "center" }}>
		<Typography color="text.secondary">No map activity in this period</Typography>
	</Box>
);

// MapStatsTrendChart renders the playtime/matches time series for the top
// maps plus the combined "Other" series as a stacked area chart.
export const MapStatsTrendChart = ({ data, metric }: { data: MapStatsChartData; metric: MapStatsMetric }) => {
	if (data.labels.length === 0 || data.series.length === 0) {
		return <EmptyChart height={chartHeight} />;
	}

	return (
		<LineChart
			height={chartHeight}
			margin={{ top: 16, right: 24, bottom: 16, left: 56 }}
			grid={{ horizontal: true }}
			xAxis={[{ scaleType: "band", data: data.labels, tickLabelStyle: { fontSize: 11 } }]}
			yAxis={[
				{ scaleType: "linear", min: 0, valueFormatter: (value: number) => formatMetricValue(value, metric) },
			]}
			series={data.series.map((s) => ({
				label: s.label,
				data: s.data,
				area: true,
				stack: "maps",
			}))}
			slotProps={{ legend: { position: "bottom", direction: "row" } }}
		/>
	);
};

// MapStatsTopChart renders the top maps by total playtime as horizontal
// bars. The backend already orders the maps; longest maps sit on top.
export const MapStatsTopChart = ({ topMaps }: { topMaps: MapStat[] }) => {
	const top = buildMapStatsTop(topMaps);
	if (top.length === 0) {
		return <EmptyChart height={chartHeight} />;
	}

	// MUI X renders band categories top-to-bottom, so reverse to put the
	// biggest map first.
	const entries = [...top].reverse();
	const height = Math.max(240, entries.length * 36 + 60);

	return (
		<BarChart
			height={height}
			layout="horizontal"
			margin={{ top: 8, right: 32, bottom: 16, left: 8 }}
			xAxis={[{ scaleType: "linear", min: 0, valueFormatter: (value: number) => `${value.toFixed(0)}h` }]}
			yAxis={[
				{
					scaleType: "band",
					data: entries.map((entry) => entry.label),
					tickLabelStyle: { fontSize: 12 },
				},
			]}
			series={[
				{
					label: "Playtime",
					data: entries.map((entry) => entry.value),
					valueFormatter: (value: number) => `${value.toFixed(1)}h`,
				},
			]}
			slotProps={{ legend: { hidden: true } }}
		/>
	);
};

// MapStatsPrefixChart renders the split across map type prefixes (cp_, pl_,
// koth_, ...) as horizontal bars for the selected metric. Workshop maps are
// already counted under their normalized prefix by the backend.
export const MapStatsPrefixChart = ({ prefixes, metric }: { prefixes: MapPrefixStat[]; metric: MapStatsMetric }) => {
	const data = buildMapStatsPrefixChart(prefixes, metric);
	if (data.length === 0) {
		return <EmptyChart height={240} />;
	}

	// MUI X renders band categories top-to-bottom, so reverse to put the
	// biggest prefix first.
	const entries = [...data].reverse();
	const height = Math.max(240, entries.length * 36 + 60);

	return (
		<BarChart
			height={height}
			layout="horizontal"
			margin={{ top: 8, right: 32, bottom: 16, left: 8 }}
			xAxis={[
				{
					scaleType: "linear",
					min: 0,
					valueFormatter: (value: number) => formatMetricValue(value, metric),
				},
			]}
			yAxis={[
				{
					scaleType: "band",
					data: entries.map((entry) => entry.label),
					tickLabelStyle: { fontSize: 12 },
				},
			]}
			series={[
				{
					label: mapStatsMetricLabel(metric),
					data: entries.map((entry) => entry.value),
					valueFormatter: (value: number) => formatMetricValue(value, metric),
				},
			]}
			slotProps={{ legend: { hidden: true } }}
		/>
	);
};
