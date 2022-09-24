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
    Legend,
    ChartOptions
} from 'chart.js';
import { Line } from 'react-chartjs-2';
import { renderDateTime } from '../util/text';
import Container from '@mui/material/Container';

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
    const options: ChartOptions = {
        responsive: true,
        plugins: {
            legend: {
                position: 'top' as const
            },
            title: {
                display: false,
                text: 'Global TF2 Server Counts'
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
                    borderColor: 'rgb(255, 99, 132)',
                    backgroundColor: 'rgba(255, 99, 132, 0.5)'
                },
                {
                    fill: true,
                    label: 'Occupied Servers',
                    data: data.map((v) => v.capacity_partial),
                    borderColor: 'rgb(246,187,60)',
                    backgroundColor: 'rgba(246,187,60, 0.5)'
                },
                {
                    fill: true,
                    label: 'Empty Servers',
                    data: data.map((v) => v.capacity_empty),
                    borderColor: 'rgb(53, 162, 235)',
                    backgroundColor: 'rgba(53, 162, 235, 0.5)'
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
