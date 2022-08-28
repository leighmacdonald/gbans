import React, { useCallback, useMemo } from 'react';
import MenuItem from '@mui/material/MenuItem';
import ListItemIcon from '@mui/material/ListItemIcon';
import AccountCircleIcon from '@mui/icons-material/AccountCircle';
import SettingsIcon from '@mui/icons-material/Settings';
import ExitToAppIcon from '@mui/icons-material/ExitToApp';
import NewspaperIcon from '@mui/icons-material/Newspaper';
import BlockIcon from '@mui/icons-material/Block';
import ReportIcon from '@mui/icons-material/Report';
import PregnantWomanIcon from '@mui/icons-material/PregnantWoman';
import ImportExportIcon from '@mui/icons-material/ImportExport';
import DnsIcon from '@mui/icons-material/Dns';
import SubjectIcon from '@mui/icons-material/Subject';
import DashboardIcon from '@mui/icons-material/Dashboard';
import StorageIcon from '@mui/icons-material/Storage';
import MenuIcon from '@mui/icons-material/Menu';
import AppBar from '@mui/material/AppBar';
import ChatIcon from '@mui/icons-material/Chat';
import ArticleIcon from '@mui/icons-material/Article';
import Avatar from '@mui/material/Avatar';
import Box from '@mui/material/Box';
import Button from '@mui/material/Button';
import Container from '@mui/material/Container';
import IconButton from '@mui/material/IconButton';
import ListItemText from '@mui/material/ListItemText';
import Menu from '@mui/material/Menu';
import Toolbar from '@mui/material/Toolbar';
import Tooltip from '@mui/material/Tooltip';
import Typography from '@mui/material/Typography';
import QueryStatsIcon from '@mui/icons-material/QueryStats';
import SupportIcon from '@mui/icons-material/Support';
import DarkModeIcon from '@mui/icons-material/DarkMode';
import LightModeIcon from '@mui/icons-material/LightMode';
import useTheme from '@mui/material/styles/useTheme';
import { useNavigate } from 'react-router-dom';
import { Flashes } from './Flashes';
import { useColourModeCtx } from '../contexts/ColourModeContext';
import { useCurrentUserCtx } from '../contexts/CurrentUserCtx';
import { PermissionLevel } from '../api';
import steamLogo from '../icons/steam_login_sm.png';
import { tf2Fonts } from '../theme';

interface menuRoute {
    to: string;
    text: string;
    icon: JSX.Element;
}

export const TopBar = () => {
    const navigate = useNavigate();
    const { currentUser } = useCurrentUserCtx();
    const user = currentUser;

    const [anchorElNav, setAnchorElNav] = React.useState<null | HTMLElement>(
        null
    );
    const [anchorElUser, setAnchorElUser] = React.useState<null | HTMLElement>(
        null
    );
    const [anchorElAdmin, setAnchorElAdmin] =
        React.useState<null | HTMLElement>(null);

    const theme = useTheme();
    const colourMode = useColourModeCtx();

    const handleOpenNavMenu = (event: React.MouseEvent<HTMLElement>) => {
        setAnchorElNav(event.currentTarget);
    };
    const handleOpenUserMenu = (event: React.MouseEvent<HTMLElement>) => {
        setAnchorElUser(event.currentTarget);
    };

    const handleOpenAdminMenu = (event: React.MouseEvent<HTMLElement>) => {
        setAnchorElAdmin(event.currentTarget);
    };

    const handleCloseNavMenu = () => {
        setAnchorElNav(null);
    };

    const handleCloseUserMenu = () => {
        setAnchorElUser(null);
    };
    const handleCloseAdminMenu = () => {
        setAnchorElAdmin(null);
    };
    const loadRoute = useCallback(
        (route: string) => {
            navigate(route);
        },
        [navigate]
    );

    const colourOpts = useMemo(() => {
        return { color: theme.palette.primary.dark };
    }, [theme.palette.primary.dark]);

    const topColourOpts = useMemo(() => {
        return { color: theme.palette.common.white };
    }, [theme.palette.common.white]);

    const menuItems: menuRoute[] = useMemo(() => {
        const items: menuRoute[] = [
            {
                to: '/',
                text: 'Dashboard',
                icon: <DashboardIcon color={'primary'} sx={topColourOpts} />
            }
            // { to: '/stats', text: 'Stats', icon: <BarChartIcon sx={{ color: '#fff' }} /> },
        ];
        if (user.ban_id <= 0) {
            items.push({
                to: '/servers',
                text: 'Servers',
                icon: <StorageIcon sx={topColourOpts} />
            });
        }
        items.push({
            to: '/wiki',
            text: 'Wiki',
            icon: <ArticleIcon sx={topColourOpts} />
        });
        if (user.ban_id <= 0) {
            items.push({
                to: '/report',
                text: 'Report',
                icon: <ReportIcon sx={topColourOpts} />
            });
        }
        if (user.permission_level >= PermissionLevel.Admin) {
            items.push({
                to: '/logs',
                text: 'Logs',
                icon: <QueryStatsIcon sx={topColourOpts} />
            });
        }
        if (user.ban_id > 0) {
            items.push({
                to: `/ban/${user.ban_id}`,
                text: 'Appeal',
                icon: <SupportIcon sx={topColourOpts} />
            });
        }
        return items;
    }, [user.ban_id, user.permission_level, topColourOpts]);

    const userItems: menuRoute[] = useMemo(
        () => [
            {
                to: `/profile/${user?.steam_id}`,
                text: 'Profile',
                icon: <AccountCircleIcon sx={colourOpts} />
            },
            // { to: '/settings', text: 'Settings', icon: <SettingsIcon /> },
            {
                to: '/logout',
                text: 'Logout',
                icon: <ExitToAppIcon sx={colourOpts} />
            }
        ],
        [colourOpts, user?.steam_id]
    );

    const adminItems: menuRoute[] = useMemo(() => {
        const items: menuRoute[] = [];
        if (user.permission_level >= PermissionLevel.Moderator) {
            items.push({
                to: '/admin/ban',
                text: 'Ban Player/Net',
                icon: <BlockIcon sx={colourOpts} />
            });
            items.push({
                to: '/admin/reports',
                text: 'Reports',
                icon: <ReportIcon sx={colourOpts} />
            });
            items.push({
                to: '/admin/chat',
                text: 'Chat History',
                icon: <ChatIcon sx={colourOpts} />
            });
            items.push({
                to: '/admin/filters',
                text: 'Filtered Words',
                icon: <SubjectIcon sx={colourOpts} />
            });
            items.push({
                to: '/admin/news',
                text: 'News',
                icon: <NewspaperIcon sx={colourOpts} />
            });
        }
        if (user.permission_level >= PermissionLevel.Admin) {
            items.push({
                to: '/admin/people',
                text: 'People',
                icon: <PregnantWomanIcon sx={colourOpts} />
            });
            items.push({
                to: '/admin/import',
                text: 'Import',
                icon: <ImportExportIcon sx={colourOpts} />
            });
            items.push({
                to: '/admin/servers',
                text: 'Servers',
                icon: <DnsIcon sx={colourOpts} />
            });
            items.push({
                to: '/admin/server_logs',
                text: 'Server Logs',
                icon: <SubjectIcon sx={colourOpts} />
            });
        }
        return items;
    }, [colourOpts, user.permission_level]);

    const renderLinkedMenuItem = (
        text: string,
        route: string,
        icon: JSX.Element
    ) => (
        <MenuItem
            onClick={() => {
                setAnchorElNav(null);
                setAnchorElUser(null);
                setAnchorElAdmin(null);
                loadRoute(route);
            }}
            key={route + text}
        >
            <ListItemIcon>{icon}</ListItemIcon>
            <ListItemText disableTypography primary={text} sx={tf2Fonts} />
        </MenuItem>
    );

    const themeIcon = useMemo(() => {
        return theme.palette.mode == 'light' ? (
            <DarkModeIcon sx={{ color: '#ada03a' }} />
        ) : (
            <LightModeIcon sx={{ color: '#ada03a' }} />
        );
    }, [theme.palette.mode]);

    return (
        <AppBar position="sticky">
            <Container maxWidth="xl">
                <Toolbar disableGutters variant="dense">
                    <Typography
                        variant="h6"
                        noWrap
                        component="div"
                        sx={{
                            mr: 2,
                            display: { xs: 'none', md: 'flex' },
                            ...tf2Fonts
                        }}
                    >
                        Uncletopia
                    </Typography>

                    <Box
                        sx={{
                            flexGrow: 1,
                            display: { xs: 'flex', md: 'none' }
                        }}
                    >
                        <IconButton
                            size="large"
                            aria-label="account of current user"
                            aria-controls="menu-appbar"
                            aria-haspopup="true"
                            onClick={handleOpenNavMenu}
                        >
                            <MenuIcon />
                        </IconButton>
                        <Menu
                            id="menu-appbar"
                            anchorEl={anchorElNav}
                            anchorOrigin={{
                                vertical: 'bottom',
                                horizontal: 'left'
                            }}
                            keepMounted
                            transformOrigin={{
                                vertical: 'top',
                                horizontal: 'left'
                            }}
                            open={Boolean(anchorElNav)}
                            onClose={handleCloseNavMenu}
                            sx={{
                                display: { xs: 'block', md: 'none' }
                            }}
                        >
                            {menuItems.map((value) => {
                                return renderLinkedMenuItem(
                                    value.text,
                                    value.to,
                                    value.icon
                                );
                            })}
                        </Menu>
                    </Box>
                    <Typography
                        variant="h6"
                        noWrap
                        component="div"
                        sx={{
                            flexGrow: 1,
                            display: { xs: 'flex', md: 'none' }
                        }}
                    >
                        Uncletopia
                    </Typography>
                    <Box
                        sx={{
                            flexGrow: 1,
                            display: { xs: 'none', md: 'flex' }
                        }}
                    >
                        {menuItems.map((value) => {
                            return renderLinkedMenuItem(
                                value.text,
                                value.to,
                                value.icon
                            );
                        })}
                    </Box>

                    <Box sx={{ flexGrow: 0 }}>
                        <>
                            <Tooltip title="Toggle BLU/RED mode">
                                <IconButton
                                    onClick={colourMode.toggleColorMode}
                                >
                                    {themeIcon}
                                </IconButton>
                            </Tooltip>
                            {!currentUser ||
                                (!user.steam_id.isValidIndividual() && (
                                    <Tooltip title="Steam Login">
                                        <Button
                                            onClick={() => {
                                                navigate('/login');
                                            }}
                                        >
                                            <img
                                                src={steamLogo}
                                                alt={'Steam Login'}
                                            />
                                        </Button>
                                    </Tooltip>
                                ))}
                            {user &&
                                user?.steam_id.isValidIndividual() &&
                                adminItems.length > 0 && (
                                    <>
                                        <Tooltip title="Mod/Admin">
                                            <IconButton
                                                sx={{
                                                    p: 0,
                                                    marginRight: '0.5rem'
                                                }}
                                                size="large"
                                                aria-label="account of current user"
                                                aria-controls="menu-appbar"
                                                aria-haspopup="true"
                                                onClick={handleOpenAdminMenu}
                                                color="inherit"
                                            >
                                                <SettingsIcon />
                                            </IconButton>
                                        </Tooltip>
                                        <Menu
                                            sx={{ mt: '45px' }}
                                            id="menu-appbar"
                                            anchorEl={anchorElAdmin}
                                            anchorOrigin={{
                                                vertical: 'top',
                                                horizontal: 'right'
                                            }}
                                            keepMounted
                                            transformOrigin={{
                                                vertical: 'top',
                                                horizontal: 'right'
                                            }}
                                            open={Boolean(anchorElAdmin)}
                                            onClose={handleCloseAdminMenu}
                                        >
                                            {adminItems.map((value) => {
                                                return renderLinkedMenuItem(
                                                    value.text,
                                                    value.to,
                                                    value.icon
                                                );
                                            })}
                                        </Menu>
                                    </>
                                )}

                            {user && user?.steam_id.isValidIndividual() && (
                                <>
                                    <Tooltip title="User Settings">
                                        <IconButton
                                            onClick={handleOpenUserMenu}
                                            sx={{ p: 0 }}
                                        >
                                            <Avatar
                                                alt={user.name}
                                                src={user.avatar}
                                            />
                                        </IconButton>
                                    </Tooltip>
                                    <Menu
                                        sx={{ mt: '45px' }}
                                        id="menu-appbar"
                                        anchorEl={anchorElUser}
                                        anchorOrigin={{
                                            vertical: 'top',
                                            horizontal: 'right'
                                        }}
                                        keepMounted
                                        transformOrigin={{
                                            vertical: 'top',
                                            horizontal: 'right'
                                        }}
                                        open={Boolean(anchorElUser)}
                                        onClose={handleCloseUserMenu}
                                    >
                                        {userItems.map((value) => {
                                            return renderLinkedMenuItem(
                                                value.text,
                                                value.to,
                                                value.icon
                                            );
                                        })}
                                    </Menu>
                                </>
                            )}
                        </>
                    </Box>
                </Toolbar>
            </Container>
            <Flashes />
        </AppBar>
    );
};
