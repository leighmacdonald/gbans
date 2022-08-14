import React, { useCallback, useMemo } from 'react';
import { useCurrentUserCtx } from '../contexts/CurrentUserCtx';
import { handleOnLogin, PermissionLevel } from '../api';
import steamLogo from '../icons/steam_login_sm.png';
import { useNavigate } from 'react-router-dom';
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
import { Flashes } from './Flashes';
import useTheme from '@mui/material/styles/useTheme';

interface menuRoute {
    to: string;
    text: string;
    icon: JSX.Element;
}

export const TopBar = () => {
    const navigate = useNavigate();
    const { currentUser } = useCurrentUserCtx();
    const perms = useMemo(
        () => parseInt(localStorage.getItem('permission_level') || '1'),
        []
    );
    const [anchorElNav, setAnchorElNav] = React.useState<null | HTMLElement>(
        null
    );
    const [anchorElUser, setAnchorElUser] = React.useState<null | HTMLElement>(
        null
    );
    const [anchorElAdmin, setAnchorElAdmin] =
        React.useState<null | HTMLElement>(null);

    const theme = useTheme();
    // const colourMode = useColourModeCtx();

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

    const menuItems: menuRoute[] = useMemo(() => {
        const items: menuRoute[] = [
            {
                to: '/',
                text: 'Dashboard',
                icon: (
                    <DashboardIcon
                        sx={{ color: theme.palette.background.default }}
                    />
                )
            },
            // { to: '/bans', text: 'Bans', icon: <BlockIcon sx={{ color: '#fff' }} /> },
            // { to: '/stats', text: 'Stats', icon: <BarChartIcon sx={{ color: '#fff' }} /> },
            {
                to: '/servers',
                text: 'Servers',
                icon: (
                    <StorageIcon
                        sx={{ color: theme.palette.background.default }}
                    />
                )
            },
            {
                to: '/report',
                text: 'Report',
                icon: (
                    <ReportIcon
                        sx={{ color: theme.palette.background.default }}
                    />
                )
            },
            // { to: '/appeal', text: 'Appeal', icon: <HistoryIcon sx={{ color: '#fff' }} /> },
            {
                to: '/wiki',
                text: 'Wiki',
                icon: (
                    <ArticleIcon
                        sx={{ color: theme.palette.background.default }}
                    />
                )
            }
        ];
        if (perms >= PermissionLevel.Admin) {
            items.push({
                to: '/logs',
                text: 'Logs',
                icon: (
                    <QueryStatsIcon
                        sx={{ color: theme.palette.background.default }}
                    />
                )
            });
        }
        if (currentUser.ban_id > 0) {
            items.push({
                to: `/ban/${currentUser.ban_id}`,
                text: 'Appeal',
                icon: (
                    <SupportIcon
                        sx={{ color: theme.palette.background.default }}
                    />
                )
            });
        }
        return items;
    }, [currentUser.ban_id, perms, theme.palette.background.default]);

    const userItems: menuRoute[] = useMemo(
        () => [
            {
                to: `/profile/${currentUser?.steam_id}`,
                text: 'Profile',
                icon: (
                    <AccountCircleIcon
                        sx={{ color: theme.palette.background.default }}
                    />
                )
            },
            // { to: '/settings', text: 'Settings', icon: <SettingsIcon /> },
            {
                to: '/logout',
                text: 'Logout',
                icon: (
                    <ExitToAppIcon
                        sx={{ color: theme.palette.background.default }}
                    />
                )
            }
        ],
        [currentUser?.steam_id, theme.palette.background.default]
    );
    const adminItems: menuRoute[] = useMemo(() => {
        const items: menuRoute[] = [];
        if (perms >= PermissionLevel.Moderator) {
            items.push({
                to: '/admin/ban',
                text: 'Ban Player/Net',
                icon: (
                    <BlockIcon
                        sx={{ color: theme.palette.background.default }}
                    />
                )
            });
            items.push({
                to: '/admin/reports',
                text: 'Reports',
                icon: (
                    <ReportIcon
                        sx={{ color: theme.palette.background.default }}
                    />
                )
            });
            items.push({
                to: '/admin/filters',
                text: 'Filtered Words',
                icon: (
                    <SubjectIcon
                        sx={{ color: theme.palette.background.default }}
                    />
                )
            });
            items.push({
                to: '/admin/news',
                text: 'News',
                icon: (
                    <NewspaperIcon
                        sx={{ color: theme.palette.background.default }}
                    />
                )
            });
        }
        if (perms >= PermissionLevel.Admin) {
            items.push({
                to: '/admin/people',
                text: 'People',
                icon: (
                    <PregnantWomanIcon
                        sx={{ color: theme.palette.background.default }}
                    />
                )
            });
            items.push({
                to: '/admin/import',
                text: 'Import',
                icon: (
                    <ImportExportIcon
                        sx={{ color: theme.palette.background.default }}
                    />
                )
            });
            items.push({
                to: '/admin/servers',
                text: 'Servers',
                icon: (
                    <DnsIcon sx={{ color: theme.palette.background.default }} />
                )
            });
            items.push({
                to: '/admin/server_logs',
                text: 'Server Logs',
                icon: (
                    <SubjectIcon
                        sx={{ color: theme.palette.background.default }}
                    />
                )
            });
        }
        return items;
    }, [perms, theme.palette.background.default]);

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
            <ListItemText primary={text} />
        </MenuItem>
    );

    return (
        <AppBar position="sticky">
            <Container maxWidth="xl">
                <Toolbar disableGutters variant="dense">
                    <Typography
                        variant="h6"
                        noWrap
                        component="div"
                        color={theme.palette.background.paper}
                        sx={{ mr: 2, display: { xs: 'none', md: 'flex' } }}
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
                            <MenuIcon
                                sx={{ color: theme.palette.background.default }}
                            />
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
                            {/*<Tooltip title="Toggle light/dark mode">*/}
                            {/*    <IconButton*/}
                            {/*        onClick={colourMode.toggleColorMode}*/}
                            {/*    >*/}
                            {/*        {theme.palette.mode == 'light' ? (*/}
                            {/*            <DarkModeIcon*/}
                            {/*                sx={{ color: '#ada03a' }}*/}
                            {/*            />*/}
                            {/*        ) : (*/}
                            {/*            <LightModeIcon*/}
                            {/*                sx={{ color: '#ada03a' }}*/}
                            {/*            />*/}
                            {/*        )}*/}
                            {/*    </IconButton>*/}
                            {/*</Tooltip>*/}
                            {!currentUser ||
                                (!currentUser.steam_id.isValidIndividual() && (
                                    <Tooltip title="Steam Login">
                                        <Button onClick={handleOnLogin}>
                                            <img
                                                src={steamLogo}
                                                alt={'Steam Login'}
                                            />
                                        </Button>
                                    </Tooltip>
                                ))}
                            {currentUser &&
                                currentUser?.steam_id.isValidIndividual() &&
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

                            {currentUser && currentUser?.steam_id && (
                                <>
                                    <Tooltip title="User Settings">
                                        <IconButton
                                            onClick={handleOpenUserMenu}
                                            sx={{ p: 0 }}
                                        >
                                            <Avatar
                                                alt={currentUser.name}
                                                src={currentUser.avatar}
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
