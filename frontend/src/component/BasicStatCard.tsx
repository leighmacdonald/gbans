import { ReactNode } from 'react';
import Button from '@mui/material/Button';
import Card from '@mui/material/Card';
import CardActions from '@mui/material/CardActions';
import CardContent from '@mui/material/CardContent';
import Stack from '@mui/material/Stack';
import Typography from '@mui/material/Typography';

interface BasicStatCardProps {
    title: string;
    value: string | number;
    desc: string;
    actionLabel?: string;
    onAction?: () => void;
    icon?: ReactNode;
}

export const BasicStatCard = ({ title, value, desc, actionLabel, onAction, icon }: BasicStatCardProps) => (
    <Card sx={{ minWidth: 275 }} variant={'outlined'}>
        <CardContent>
            <Stack direction={'row'} spacing={1}>
                {icon}
                <Typography
                    sx={{ fontSize: 14 }}
                    color="text.secondary"
                    //gutterBottom={true}
                >
                    {title}
                </Typography>
            </Stack>
            <Typography variant="h1" component="div">
                {value}
            </Typography>
            {/*<Typography sx={{ mb: 1.5 }} color="text.secondary">*/}
            {/*    adjective*/}
            {/*</Typography>*/}
            <Typography variant="body2">{desc}</Typography>
        </CardContent>
        {actionLabel && (
            <CardActions>
                <Button size="small" onClick={onAction}>
                    {actionLabel}
                </Button>
            </CardActions>
        )}
    </Card>
);
