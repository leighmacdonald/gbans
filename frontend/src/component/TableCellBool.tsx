import CheckIcon from '@mui/icons-material/Check';
import CloseIcon from '@mui/icons-material/Close';

export const TableCellBool = ({ enabled }: { enabled: boolean }) => {
    return <div>{enabled ? <CheckIcon color={'success'} /> : <CloseIcon color={'error'} />}</div>;
};
