import React, { useMemo } from 'react';
import { GlobalTF2StatSnapshot } from '../api';
import {
    Chart as ChartJS,
    CategoryScale,
    LinearScale,
    PointElement,
    LineElement,
    Title,
    Filler,
    Tooltip,
    Legend
} from 'chart.js';
import { Line } from 'react-chartjs-2';
import { renderDateTime } from '../util/text';
import Container from '@mui/material/Container';
import { Colors, ColorsTrans, makeChartOpts } from '../util/ui';

ChartJS.register(
    CategoryScale,
    LinearScale,
    PointElement,
    LineElement,
    Title,
    Filler,
    Tooltip,
    Legend
);

export interface ServerStatsChartProps {
    data: GlobalTF2StatSnapshot[];
}

export const ServerStatsChart = ({ data }: ServerStatsChartProps) => {
    const options = makeChartOpts('Global TF2 Server Counts');
    const chartData = useMemo(() => {
        return {
            labels: data.map((d) => renderDateTime(d.created_on)),
            datasets: [
                {
                    fill: true,
                    label: 'Full Servers',
                    data: data.map((v) => v.capacity_full),
                    borderColor: Colors[0],
                    backgroundColor: ColorsTrans[0]
                },
                {
                    fill: true,
                    label: 'Occupied Servers',
                    data: data.map((v) => v.capacity_partial),
                    borderColor: Colors[1],
                    backgroundColor: ColorsTrans[1]
                },
                {
                    fill: true,
                    label: 'Empty Servers',
                    data: data.map((v) => v.capacity_empty),
                    borderColor: Colors[2],
                    backgroundColor: ColorsTrans[2]
                }
            ]
        };
    }, [data]);

    return (
        <Container sx={{ padding: 2 }}>
            {chartData ? <Line options={options} data={chartData} /> : <></>}
        </Container>
    );
};
