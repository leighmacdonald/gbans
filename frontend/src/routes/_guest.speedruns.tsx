import ElectricBoltIcon from '@mui/icons-material/ElectricBolt';
import EmojiEventsIcon from '@mui/icons-material/EmojiEvents';
import HistoryIcon from '@mui/icons-material/History';
import Typography from '@mui/material/Typography';
import Grid from '@mui/material/Unstable_Grid2';
import { createFileRoute, useLoaderData } from '@tanstack/react-router';
import { z } from 'zod';
import { getSpeedrunsOverall, SpeedrunResult } from '../api';
import { ContainerWithHeader } from '../component/ContainerWithHeader.tsx';
import { Title } from '../component/Title';

const demosSchema = z.object({
    map_name: z.string().optional(),
    server_id: z.number().optional()
});

export const Route = createFileRoute('/_guest/speedruns')({
    component: Speedruns,
    // beforeLoad: () => {
    //     TODO
    //     checkFeatureEnabled('1ku');
    // },
    validateSearch: (search) => demosSchema.parse(search),
    loader: async ({ context }) => {
        try {
            return (
                (await context.queryClient.ensureQueryData({
                    queryKey: ['speedruns_overall'],
                    queryFn: getSpeedrunsOverall
                })) ?? []
            );
        } catch {
            return [
                { map_name: 'pl_basdfasdf_adfsaf' },
                { map_name: 'pl_basdfasdf_adfsaf' },
                { map_name: 'pl_basdfasdf_adfsaf' },
                { map_name: 'pl_basdfasdf_adfsaf' },
                { map_name: 'pl_basdfasdf_adfsaf' },
                { map_name: 'pl_basdfasdf_adfsaf' },
                { map_name: 'pl_basdfasdf_adfsaf' },
                { map_name: 'pl_basdfasdf_adfsaf' },
                { map_name: 'pl_basdfasdf_adfsaf' },
                { map_name: 'pl_basdfasdf_adfsaf' },
                { map_name: 'pl_basdfasdf_adfsaf' },
                { map_name: 'pl_basdfasdf_adfsaf' },
                { map_name: 'pl_basdfasdf_adfsaf' },
                { map_name: 'pl_basdfasdf_adfsaf' },
                { map_name: 'pl_basdfasdf_adfsaf' },
                { map_name: 'pl_basdfasdf_adfsaf' },
                { map_name: 'pl_basdfasdf_adfsaf' },
                { map_name: 'pl_basdfasdf_adfsaf' },
                { map_name: 'pl_basdfasdf_adfsaf' },
                { map_name: 'pl_basdfasdf_adfsaf' },
                { map_name: 'pl_basdfasdf_adfsaf' }
            ];
        }
    }
});

function Speedruns() {
    // const navigate = useNavigate({ from: Route.fullPath });
    // const search = Route.useSearch();
    //
    const speedruns = useLoaderData({ from: '/_guest/speedruns' }) as SpeedrunResult[];

    return (
        <>
            <Title>Speedrun Overall Results</Title>
            <Grid container spacing={2}>
                <Grid xs={12}>
                    <ContainerWithHeader title={'Speedruns'} iconLeft={<ElectricBoltIcon />}>
                        <Typography>
                            These are the overall results for the speedruns. Speedruns are automatically started once
                            the map starts. For a player to count in the overall participants, they must have played a
                            minimum of 15% of the runs lengths as played time in that specific map.
                        </Typography>
                    </ContainerWithHeader>
                </Grid>

                <Grid xs={12}>
                    <ContainerWithHeader title={'Most Recent Changes'} iconLeft={<HistoryIcon />}>
                        Table of changes...
                    </ContainerWithHeader>
                </Grid>

                {speedruns.map((sr) => {
                    return (
                        <Grid xs={6} md={4}>
                            <ContainerWithHeader title={sr.map_name} iconLeft={<EmojiEventsIcon />}>
                                {sr.map_name}
                            </ContainerWithHeader>
                        </Grid>
                    );
                })}
            </Grid>
        </>
    );
}
