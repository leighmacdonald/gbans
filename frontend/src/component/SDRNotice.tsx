import Paper from '@mui/material/Paper';
import Typography from '@mui/material/Typography';

export const SDRNotice = () => (
    <Paper elevation={1}>
        <Typography variant={'body1'} padding={1}>
            Note that when using Valves SDR (Steam Datagram Relay), IP bans are effective non-functional as players are
            allocated ips dynamically within a shared pool (169.254.0.0/16).
        </Typography>
    </Paper>
);
