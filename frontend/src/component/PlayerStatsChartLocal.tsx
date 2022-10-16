import React, { useMemo } from 'react';
import { LocalTF2StatSnapshot } from '../api';
import {
    Chart as ChartJS,
    CategoryScale,
    LinearScale,
    PointElement,
    LineElement,
    Title,
    Tooltip,
    Legend,
    Filler
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

export const PlayerStatsChartLocal = ({
    data
}: {
    data: LocalTF2StatSnapshot[];
}) => {
    const options = makeChartOpts('Local TF2 Player Counts');
    const chartData = useMemo(() => {
        return {
            labels: data.map((d) => renderDateTime(d.created_on)),
            datasets: [
                {
                    fill: true,
                    label: 'Players',
                    data: data.map((v) => v.players),
                    borderColor: Colors[0],
                    backgroundColor: ColorsTrans[0]
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
