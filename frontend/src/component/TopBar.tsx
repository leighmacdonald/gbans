import React, { useState } from 'react';
import { useCurrentUserCtx } from '../contexts/CurrentUserCtx';
import { handleOnLogin, PermissionLevel } from '../util/api';
import { GLink } from './GLink';
import steamLogo from '../icons/steam_login_sm.png';
import { useNavigate } from 'react-router-dom';
import MenuItem from '@mui/material/MenuItem';
import ListItemIcon from '@mui/material/ListItemIcon';
import AccountCircleIcon from '@mui/icons-material/AccountCircle';
import SettingsIcon from '@mui/icons-material/Settings';
import ExitToAppIcon from '@mui/icons-material/ExitToApp';
import BlockIcon from '@mui/icons-material/Block';
import ReportIcon from '@mui/icons-material/Report';
import PregnantWomanIcon from '@mui/icons-material/PregnantWoman';
import ImportExportIcon from '@mui/icons-material/ImportExport';
import DnsIcon from '@mui/icons-material/Dns';
import SubjectIcon from '@mui/icons-material/Subject';
import DashboardIcon from '@mui/icons-material/Dashboard';
import StorageIcon from '@mui/icons-material/Storage';
import HistoryIcon from '@mui/icons-material/History';
import MoreIcon from '@mui/icons-material/More';
import SearchIcon from '@mui/icons-material/Search';
import NotificationsIcon from '@mui/icons-material/Notifications';

import {
    AppBar,
    Avatar,
    Badge,
    Button,
    Divider,
    IconButton,
    InputBase,
    ListItemText,
    Menu,
    Toolbar,
    Typography
} from '@mui/material';
//
// const useStyles = makeStyles((theme: Theme) => ({
//     grow: {
//         flexGrow: 1
//     },
//     menuButton: {
//         marginRight: theme.spacing(2)
//     },
//     title: {
//         display: 'none',
//         [theme.breakpoints.up('sm')]: {
//             display: 'block'
//         }
//     },
//     search: {
//         position: 'relative',
//         borderRadius: theme.shape.borderRadius,
//         backgroundColor: alpha(theme.palette.common.white, 0.15),
//         '&:hover': {
//             backgroundColor: alpha(theme.palette.common.white, 0.25)
//         },
//         marginRight: theme.spacing(2),
//         marginLeft: 0,
//         width: '100%',
//         [theme.breakpoints.up('sm')]: {
//             marginLeft: theme.spacing(3),
//             width: 'auto'
//         }
//     },
//     searchIcon: {
//         padding: theme.spacing(0, 2),
//         height: '100%',
//         position: 'absolute',
//         pointerEvents: 'none',
//         display: 'flex',
//         alignItems: 'center',
//         justifyContent: 'center'
//     },
//     inputRoot: {
//         color: 'inherit'
//     },
//     inputInput: {
//         padding: theme.spacing(1, 1, 1, 0),
//         // vertical padding + font size from searchIcon
//         paddingLeft: `calc(1em + ${theme.spacing(4)}px)`,
//         transition: theme.transitions.create('width'),
//         width: '100%',
//         [theme.breakpoints.up('md')]: {
//             width: '20ch'
//         }
//     },
//     sectionDesktop: {
//         display: 'none',
//         [theme.breakpoints.up('md')]: {
//             display: 'flex'
//         }
//     },
//     sectionMobile: {
//         display: 'flex',
//         [theme.breakpoints.up('md')]: {
//             display: 'none'
//         }
//     },
//     root: {
//         display: 'flex'
//     },
//     appBar: {
//         zIndex: theme.zIndex.drawer + 1,
//         transition: theme.transitions.create(['width', 'margin'], {
//             easing: theme.transitions.easing.sharp,
//             duration: theme.transitions.duration.leavingScreen
//         })
//     }
// }));

export const TopBar = (): JSX.Element => {
    const navigate = useNavigate();
    const [anchorProfileMenuEl, setAnchorProfileMenuEl] =
        useState<Element | null>(null);
    const [anchorAdminMenuEl, setAnchorAdminMenuEl] = useState<Element | null>(
        null
    );
    const [mobileMoreAnchorEl, setMobileMoreAnchorEl] =
        useState<Element | null>(null);

    const isProfileMenuOpen = Boolean(anchorProfileMenuEl);
    const isAdminMenuOpen = Boolean(anchorAdminMenuEl);
    const isMobileMenuOpen = Boolean(mobileMoreAnchorEl);
    const { currentUser } = useCurrentUserCtx();

    const handleAdminMenuOpen = (event: React.MouseEvent) => {
        setAnchorAdminMenuEl(event.currentTarget);
    };
    const handleAdminMenuClose = () => {
        setAnchorAdminMenuEl(null);
        handleMobileMenuClose();
    };

    const handleProfileMenuOpen = (event: React.MouseEvent) => {
        setAnchorProfileMenuEl(event.currentTarget);
    };

    const handleMobileMenuClose = () => {
        setMobileMoreAnchorEl(null);
    };

    const handleProfileMenuClose = () => {
        setAnchorProfileMenuEl(null);
        handleMobileMenuClose();
    };

    const handleMobileMenuOpen = (event: React.MouseEvent) => {
        setMobileMoreAnchorEl(event.currentTarget);
    };

    const menuId = 'primary-search-account-menu';
    const adminMenuId = 'admin-menu';

    const loadRoute = (route: string) => {
        navigate(route);
        handleProfileMenuClose();
        handleAdminMenuClose();
        handleMobileMenuClose();
    };

    const renderLinkedMenuItem = (
        text: string,
        route: string,
        icon: JSX.Element
    ) => (
        <MenuItem onClick={() => loadRoute(route)}>
            <ListItemIcon>{icon}</ListItemIcon>
            <ListItemText primary={text} />
        </MenuItem>
    );

    const renderProfileMenu = (
        <Menu
            anchorEl={anchorProfileMenuEl}
            anchorOrigin={{ vertical: 'top', horizontal: 'right' }}
            id={menuId}
            keepMounted
            transformOrigin={{ vertical: 'top', horizontal: 'right' }}
            open={isProfileMenuOpen}
            onClose={handleProfileMenuClose}
        >
            {renderLinkedMenuItem('Profile', '/profile', <AccountCircleIcon />)}
            {renderLinkedMenuItem('Settings', '/settings', <SettingsIcon />)}
            <Divider light />
            {renderLinkedMenuItem('Logout', '/logout', <ExitToAppIcon />)}
        </Menu>
    );
    const perms = parseInt(localStorage.getItem('permission_level') || '1');
    const renderAdminMenu = (
        <Menu
            anchorEl={anchorAdminMenuEl}
            anchorOrigin={{ vertical: 'top', horizontal: 'right' }}
            id={adminMenuId}
            keepMounted
            transformOrigin={{ vertical: 'top', horizontal: 'right' }}
            open={isAdminMenuOpen}
            onClose={handleAdminMenuClose}
        >
            {perms >= PermissionLevel.Moderator &&
                renderLinkedMenuItem(
                    'Ban Player/Net',
                    '/admin/ban',
                    <BlockIcon />
                )}
            {perms >= PermissionLevel.Moderator &&
                renderLinkedMenuItem(
                    'Reports',
                    '/admin/reports',
                    <ReportIcon />
                )}
            {perms >= PermissionLevel.Admin &&
                renderLinkedMenuItem(
                    'People',
                    '/admin/people',
                    <PregnantWomanIcon />
                )}
            {perms >= PermissionLevel.Admin &&
                renderLinkedMenuItem(
                    'Import',
                    '/admin/import',
                    <ImportExportIcon />
                )}
            {perms >= PermissionLevel.Moderator &&
                renderLinkedMenuItem(
                    'Filtered Words',
                    '/admin/filters',
                    <SubjectIcon />
                )}
            {perms >= PermissionLevel.Admin &&
                renderLinkedMenuItem('Servers', '/admin/servers', <DnsIcon />)}
            {perms >= PermissionLevel.Admin &&
                renderLinkedMenuItem(
                    'Server Logs',
                    '/admin/server_logs',
                    <SubjectIcon />
                )}
        </Menu>
    );

    const mobileMenuId = 'primary-search-account-menu-mobile';
    const renderMobileMenu = (
        <Menu
            anchorEl={mobileMoreAnchorEl}
            anchorOrigin={{ vertical: 'top', horizontal: 'right' }}
            id={mobileMenuId}
            keepMounted
            transformOrigin={{ vertical: 'top', horizontal: 'right' }}
            open={isMobileMenuOpen}
            onClose={handleMobileMenuClose}
        >
            <MenuItem>
                <IconButton
                    aria-label="show 0 new notifications"
                    color="inherit"
                >
                    <Badge badgeContent={0} color="secondary">
                        <NotificationsIcon />
                    </Badge>
                </IconButton>
                <p>Notifications</p>
            </MenuItem>
            <MenuItem onClick={handleProfileMenuOpen}>
                <IconButton
                    aria-label="account of current user"
                    aria-controls="primary-search-account-menu"
                    aria-haspopup="true"
                    color="inherit"
                >
                    <AccountCircleIcon />
                </IconButton>
                <p>Profile</p>
            </MenuItem>
        </Menu>
    );

    return (
        <>
            <div>
                <AppBar position="fixed">
                    <Toolbar>
                        <Typography variant="h6" noWrap>
                            <GLink
                                to={'/'}
                                primary={'Dashboard'}
                                icon={<DashboardIcon />}
                            />
                        </Typography>
                        <GLink
                            to={'/bans'}
                            primary={'Bans'}
                            icon={<BlockIcon />}
                        />
                        <GLink
                            to={'/servers'}
                            primary={'Servers'}
                            icon={<StorageIcon />}
                        />
                        <GLink
                            to={'/appeal'}
                            primary={'Appeal'}
                            icon={<HistoryIcon />}
                        />
                        <div>
                            <div>
                                <SearchIcon />
                            </div>
                            <InputBase
                                placeholder="Search…"
                                inputProps={{ 'aria-label': 'search' }}
                            />
                        </div>
                        <div />
                        <div>
                            {!currentUser.player ||
                                (currentUser?.player.steam_id === '' && (
                                    <Button onClick={handleOnLogin}>
                                        <img
                                            src={steamLogo}
                                            alt={'Steam Login'}
                                        />
                                    </Button>
                                ))}
                            {currentUser?.player.steam_id != '' && (
                                <>
                                    <IconButton
                                        aria-label="no notifications"
                                        color="inherit"
                                    >
                                        <Badge badgeContent={0}>
                                            <NotificationsIcon />
                                        </Badge>
                                    </IconButton>
                                    {perms >= PermissionLevel.Moderator && (
                                        <IconButton
                                            edge="end"
                                            aria-label="admin menu"
                                            aria-controls={menuId}
                                            aria-haspopup="true"
                                            onClick={handleAdminMenuOpen}
                                            color="inherit"
                                        >
                                            <SettingsIcon />
                                        </IconButton>
                                    )}
                                    <IconButton
                                        edge="end"
                                        aria-label="account of current user"
                                        aria-controls={menuId}
                                        aria-haspopup="true"
                                        onClick={handleProfileMenuOpen}
                                        color="inherit"
                                    >
                                        <Avatar
                                            alt={currentUser.player.personaname}
                                            src={currentUser.player.avatar}
                                        />
                                    </IconButton>
                                </>
                            )}
                        </div>
                        <div>
                            <IconButton
                                aria-label="show more"
                                aria-controls={mobileMenuId}
                                aria-haspopup="true"
                                onClick={handleMobileMenuOpen}
                                color="inherit"
                            >
                                <MoreIcon />
                            </IconButton>
                        </div>
                    </Toolbar>
                </AppBar>

                {perms >= PermissionLevel.Moderator && renderAdminMenu}
                {renderMobileMenu}
                {renderProfileMenu}
            </div>
        </>
    );
};
