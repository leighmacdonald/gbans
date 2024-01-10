import React from 'react';
import { useNavigate } from 'react-router-dom';
import HistoryIcon from '@mui/icons-material/History';
import ReportIcon from '@mui/icons-material/Report';
import ReportGmailerrorredIcon from '@mui/icons-material/ReportGmailerrorred';
import SettingsSuggestIcon from '@mui/icons-material/SettingsSuggest';
import { Divider, IconButton } from '@mui/material';
import ListItemIcon from '@mui/material/ListItemIcon';
import ListItemText from '@mui/material/ListItemText';
import Menu from '@mui/material/Menu';
import MenuItem from '@mui/material/MenuItem';
import {
    sessionKeyReportPersonMessageIdName,
    sessionKeyReportSteamID
} from '../api';

interface ChatContextMenuProps {
    person_message_id: number;
    flagged: boolean;
    steamId: string;
}

export const ChatContextMenu = ({
    person_message_id,
    flagged,
    steamId
}: ChatContextMenuProps) => {
    const navigate = useNavigate();

    const [anchorEl, setAnchorEl] = React.useState<null | HTMLElement>(null);
    const open = Boolean(anchorEl);
    const handleClick = (event: React.MouseEvent<HTMLElement>) => {
        setAnchorEl(event.currentTarget);
    };
    const handleClose = () => {
        setAnchorEl(null);
    };

    const onClickReport = () => {
        sessionStorage.setItem(
            sessionKeyReportPersonMessageIdName,
            `${person_message_id}`
        );
        sessionStorage.setItem(sessionKeyReportSteamID, steamId);
        navigate('/report');
        handleClose();
    };

    return (
        <>
            <IconButton onClick={handleClick} size={'small'}>
                <SettingsSuggestIcon color={'info'} />
            </IconButton>
            <Menu
                id="chat-msg-menu"
                anchorEl={anchorEl}
                open={open}
                onClose={handleClose}
                anchorOrigin={{
                    vertical: 'top',
                    horizontal: 'left'
                }}
                transformOrigin={{
                    vertical: 'top',
                    horizontal: 'left'
                }}
            >
                <MenuItem onClick={onClickReport} disabled={flagged}>
                    <ListItemIcon>
                        <ReportIcon fontSize="small" color={'error'} />
                    </ListItemIcon>
                    <ListItemText>Create Report (Full)</ListItemText>
                </MenuItem>
                <MenuItem onClick={onClickReport} disabled={true}>
                    <ListItemIcon>
                        <ReportGmailerrorredIcon
                            fontSize="small"
                            color={'error'}
                        />
                    </ListItemIcon>
                    <ListItemText>Create Report (1-Click)</ListItemText>
                </MenuItem>
                <Divider />
                <MenuItem onClick={handleClose} disabled={true}>
                    <ListItemIcon>
                        <HistoryIcon fontSize="small" />
                    </ListItemIcon>
                    <ListItemText>Message Context</ListItemText>
                </MenuItem>
            </Menu>
        </>
    );
};
