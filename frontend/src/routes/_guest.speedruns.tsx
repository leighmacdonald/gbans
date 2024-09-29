import Typography from '@mui/material/Typography';
import Grid from '@mui/material/Unstable_Grid2';
import { createFileRoute } from '@tanstack/react-router';
import { z } from 'zod';
import { apiGetServers } from '../api';
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
        const unsorted = await context.queryClient.ensureQueryData({
            queryKey: ['serversSimple'],
            queryFn: apiGetServers
        });

        return {
            servers: unsorted.sort((a, b) => {
                if (a.server_name > b.server_name) {
                    return 1;
                }
                if (a.server_name < b.server_name) {
                    return -1;
                }
                return 0;
            })
        };
    }
});

function Speedruns() {
    // const navigate = useNavigate({ from: Route.fullPath });
    // const search = Route.useSearch();
    //
    // const { data: demos, isLoading } = useQuery({
    //     queryKey: ['demos'],
    //     queryFn: apiGetDemos
    // });

    return (
        <>
            <Title>SourceTV</Title>
            <Grid container spacing={2}>
                <Grid xs={12}>
                    <Typography>Speedruns</Typography>
                </Grid>
            </Grid>
        </>
    );
}
