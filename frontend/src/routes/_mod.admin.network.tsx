import Link from '@mui/material/Link';
import Paper from '@mui/material/Paper';
import Typography from '@mui/material/Typography';
import Grid from '@mui/material/Unstable_Grid2';
import Box from '@mui/system/Box';
import { createFileRoute } from '@tanstack/react-router';

export const Route = createFileRoute('/_mod/admin/network')({
    component: AdminNetwork
});

function AdminNetwork() {
    return (
        <Grid container spacing={2}>
            <Grid xs={6} md={6}>
                <Link href={`/admin/network/ip_hist`}>
                    <Paper component={Box} padding={2}>
                        <Typography variant="h5">Player IP History</Typography>
                        <Typography variant="body1">Query IPs a player has used</Typography>
                    </Paper>
                </Link>
            </Grid>
            <Grid xs={6} md={6}>
                <Link href={`/admin/network/players_by_ip`}>
                    <Paper component={Box} padding={2}>
                        <Typography variant="h5">Find Players By IP</Typography>
                        <Typography variant="body1">Find all players who fall under a particular IP or CIDR range.</Typography>
                    </Paper>
                </Link>
            </Grid>
            <Grid xs={6} md={6}>
                <Link href={`/admin/network/ip_info`}>
                    <Paper component={Box} padding={2}>
                        <Typography variant="h5">Network Info</Typography>
                        <Typography variant="body1">
                            Get higher level info about a particular ip. This include location, proxy info, ASN network blocks that the IP
                            belongs to
                        </Typography>
                    </Paper>
                </Link>
            </Grid>
            <Grid xs={6} md={6}>
                <Link href={`/admin/network/cidr_blocks`}>
                    <Paper component={Box} padding={2}>
                        <Typography variant="h5">External CIDR Bans</Typography>
                        <Typography variant="body1">
                            Used for banning large range of address blocks using 3rd party URL sources. Response should be in the format of
                            1 cidr address per line. Invalid lines are discarded. Use the whitelist to override blocked addresses you want
                            to allow.
                        </Typography>
                    </Paper>
                </Link>
            </Grid>
        </Grid>
    );
}
