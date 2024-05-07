import Typography from '@mui/material/Typography';

export const TableHeadingCell = ({ name }: { name: string }) => {
    return (
        <Typography align={'left'} padding={0} fontWeight={700}>
            {name}
        </Typography>
    );
};
