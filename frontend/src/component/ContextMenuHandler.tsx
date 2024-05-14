import { ReactNode, useMemo } from 'react';
import ListItemIcon from '@mui/material/ListItemIcon';
import ListItemText from '@mui/material/ListItemText';
import Menu from '@mui/material/Menu';
import MenuItem from '@mui/material/MenuItem';
import { bindMenu, PopupState } from 'material-ui-popup-state/hooks';

export interface ChatContextMenuProps {
    handlers: ClickHandler[];
    popupState: PopupState;
}

export type ClickHandler = {
    label: string;
    icon: ReactNode;
    onClick: () => void;
};

export const ContextMenuHandler = ({ popupState, handlers }: ChatContextMenuProps) => {
    const menu = useMemo(() => {
        return handlers.map((h) => {
            return (
                <MenuItem onClick={h.onClick} key={`mi-${h.label}`}>
                    <ListItemIcon>{h.icon}</ListItemIcon>
                    <ListItemText>{h.label}</ListItemText>
                </MenuItem>
            );
        });
    }, [handlers]);

    return (
        <Menu
            id="ctx-menu"
            {...bindMenu(popupState)}
            anchorOrigin={{
                vertical: 'top',
                horizontal: 'left'
            }}
            transformOrigin={{
                vertical: 'top',
                horizontal: 'left'
            }}
        >
            {menu}
        </Menu>
    );
};
