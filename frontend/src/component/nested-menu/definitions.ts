import { SxProps } from '@mui/material/styles';

export interface MenuItemData {
    uid?: string;
    href?: string;
    label?: React.ReactNode;
    leftIcon?: React.ReactNode;
    rightIcon?: React.ReactNode;
    callback?: (
        event: React.MouseEvent<HTMLElement>,
        item: MenuItemData
    ) => void;
    items?: MenuItemData[];
    disabled?: boolean;
    sx?: SxProps;
}
