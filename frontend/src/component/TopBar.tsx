import React, { useState } from 'react';
import { alpha, makeStyles, Theme } from '@material-ui/core/styles';
import AppBar from '@material-ui/core/AppBar';
import Toolbar from '@material-ui/core/Toolbar';
import IconButton from '@material-ui/core/IconButton';
import Typography from '@material-ui/core/Typography';
import InputBase from '@material-ui/core/InputBase';
import Badge from '@material-ui/core/Badge';
import MenuItem from '@material-ui/core/MenuItem';
import Menu from '@material-ui/core/Menu';
import SearchIcon from '@material-ui/icons/Search';
import AccountCircle from '@material-ui/icons/AccountCircle';
import AccountCircleIcon from '@material-ui/icons/AccountCircle';
import NotificationsIcon from '@material-ui/icons/Notifications';
import MoreIcon from '@material-ui/icons/MoreVert';
import { useCurrentUserCtx } from '../contexts/CurrentUserCtx';
import { handleOnLogin, PermissionLevel } from '../util/api';
import {
    Avatar,
    Button,
    Divider,
    ListItemIcon,
    ListItemText
} from '@material-ui/core';
import SettingsIcon from '@material-ui/icons/Settings';
import DashboardIcon from '@material-ui/icons/Dashboard';
import HistoryIcon from '@material-ui/icons/History';
import StorageIcon from '@material-ui/icons/Storage';
import BlockIcon from '@material-ui/icons/Block';
import ReportIcon from '@material-ui/icons/Report';
import PregnantWomanIcon from '@material-ui/icons/PregnantWoman';
import ImportExportIcon from '@material-ui/icons/ImportExport';
import SpellcheckIcon from '@material-ui/icons/Spellcheck';
import DnsIcon from '@material-ui/icons/Dns';
import SubjectIcon from '@material-ui/icons/Subject';
import ExitToAppIcon from '@material-ui/icons/ExitToApp';
import { GLink } from './GLink';
import steamLogo from '../icons/steam_login_sm.png';
import { useNavigate } from 'react-router-dom';

const useStyles = makeStyles((theme: Theme) => ({
    grow: {
        flexGrow: 1
    },
    menuButton: {
        marginRight: theme.spacing(2)
    },
    title: {
        display: 'none',
        [theme.breakpoints.up('sm')]: {
            display: 'block'
        }
    },
    search: {
        position: 'relative',
        borderRadius: theme.shape.borderRadius,
        backgroundColor: alpha(theme.palette.common.white, 0.15),
        '&:hover': {
            backgroundColor: alpha(theme.palette.common.white, 0.25)
        },
        marginRight: theme.spacing(2),
        marginLeft: 0,
        width: '100%',
        [theme.breakpoints.up('sm')]: {
            marginLeft: theme.spacing(3),
            width: 'auto'
        }
    },
    searchIcon: {
        padding: theme.spacing(0, 2),
        height: '100%',
        position: 'absolute',
        pointerEvents: 'none',
        display: 'flex',
        alignItems: 'center',
        justifyContent: 'center'
    },
    inputRoot: {
        color: 'inherit'
    },
    inputInput: {
        padding: theme.spacing(1, 1, 1, 0),
        // vertical padding + font size from searchIcon
        paddingLeft: `calc(1em + ${theme.spacing(4)}px)`,
        transition: theme.transitions.create('width'),
        width: '100%',
        [theme.breakpoints.up('md')]: {
            width: '20ch'
        }
    },
    sectionDesktop: {
        display: 'none',
        [theme.breakpoints.up('md')]: {
            display: 'flex'
        }
    },
    sectionMobile: {
        display: 'flex',
        [theme.breakpoints.up('md')]: {
            display: 'none'
        }
    },
    root: {
        display: 'flex'
    },
    appBar: {
        zIndex: theme.zIndex.drawer + 1,
        transition: theme.transitions.create(['width', 'margin'], {
            easing: theme.transitions.easing.sharp,
            duration: theme.transitions.duration.leavingScreen
        })
    }
}));

export const TopBar = (): JSX.Element => {
    const classes = useStyles();
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
                    <SpellcheckIcon />
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
                    <AccountCircle />
                </IconButton>
                <p>Profile</p>
            </MenuItem>
        </Menu>
    );

    return (
        <>
            <div className={classes.grow}>
                <AppBar position="fixed">
                    <Toolbar>
                        <Typography
                            className={classes.title}
                            variant="h6"
                            noWrap
                        >
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
                        <div className={classes.search}>
                            <div className={classes.searchIcon}>
                                <SearchIcon />
                            </div>
                            <InputBase
                                placeholder="Search…"
                                classes={{
                                    root: classes.inputRoot,
                                    input: classes.inputInput
                                }}
                                inputProps={{ 'aria-label': 'search' }}
                            />
                        </div>
                        <div className={classes.grow} />
                        <div className={classes.sectionDesktop}>
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
                        <div className={classes.sectionMobile}>
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
