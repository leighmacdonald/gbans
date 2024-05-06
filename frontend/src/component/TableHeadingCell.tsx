import Typography from '@mui/material/Typography';

export const TableHeadingCell = ({ name }: { name: string }) => {
    return (
        <Typography align={'left'} padding={0}>
            {name}
        </Typography>
    );
};
