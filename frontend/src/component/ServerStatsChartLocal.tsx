import React, { useMemo } from 'react';
import { LocalTF2StatSnapshot } from '../api';
import {
    Chart as ChartJS,
    CategoryScale,
    LinearScale,
    PointElement,
    LineElement,
    Title,
    Filler,
    Tooltip,
    Legend,
    ChartOptions
} from 'chart.js';
import { Line } from 'react-chartjs-2';
import { renderDateTime } from '../util/text';
import Container from '@mui/material/Container';
import { Colors, ColorsTrans } from '../util/ui';

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
    data: LocalTF2StatSnapshot[];
}

export const ServerStatsChartLocal = ({ data }: ServerStatsChartProps) => {
    const options: ChartOptions = {
        responsive: true,
        plugins: {
            legend: {
                position: 'top' as const
            },
            title: {
                display: false,
                text: 'Local TF2 Server Counts'
            }
        }
    };

    const labels = useMemo(() => {
        return data.map((d) => renderDateTime(d.created_on));
    }, [data]);

    const chartData = useMemo(() => {
        return {
            labels,
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
    }, [data, labels]);

    return (
        <Container sx={{ padding: 2 }}>
            {chartData ? <Line options={options} data={chartData} /> : <></>}
        </Container>
    );
};
