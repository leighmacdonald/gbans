import {
    createStyles,
    TableCell,
    TableRow,
    withStyles
} from '@material-ui/core';
import { Theme } from '@material-ui/core/styles';

export const StyledTableCell = withStyles((theme: Theme) =>
    createStyles({
        head: {
            backgroundColor: theme.palette.primary.main,
            color: theme.palette.common.white
        },
        body: {
            fontSize: 14
        }
    })
)(TableCell);

export const StyledTableRow = withStyles((theme: Theme) =>
    createStyles({
        root: {
            '&:nth-of-type(odd)': {
                backgroundColor: theme.palette.action.hover
            }
        }
    })
)(TableRow);
