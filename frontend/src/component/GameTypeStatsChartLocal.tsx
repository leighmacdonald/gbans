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
import uniq from 'lodash-es/uniq';
import flatten from 'lodash-es/flatten';
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

export const GameTypeStatsChartLocal = ({
    data
}: {
    data: LocalTF2StatSnapshot[];
}) => {
    const options = makeChartOpts('Local TF2 Game Types');
    const chartData = useMemo(() => {
        const mapKeys = uniq(
            flatten(data.map((d) => Object.keys(d.map_types)))
        );
        return {
            labels: data.map((d) => renderDateTime(d.created_on)),
            datasets: mapKeys.map((key, index) => {
                return {
                    fill: true,
                    label: key,
                    data: data.map((v) => v.map_types[key] ?? 0),
                    borderColor: Colors[index],
                    backgroundColor: ColorsTrans[index]
                };
            })
        };
    }, [data]);

    return (
        <Container sx={{ padding: 2 }}>
            {chartData ? <Line options={options} data={chartData} /> : <></>}
        </Container>
    );
};
