import { useMemo } from 'react';
import Tooltip from '@mui/material/Tooltip';
import Typography from '@mui/material/Typography';

export const TableHeadingCell = ({ name, tooltip }: { name: string; tooltip?: string }) => {
    return useMemo(() => {
        const childElement = (
            <Typography align={'left'} padding={0} fontWeight={700}>
                {name}
            </Typography>
        );
        if (tooltip) {
            return <Tooltip title={tooltip}>{childElement}</Tooltip>;
        }
        return name;
    }, [name, tooltip]);
};
