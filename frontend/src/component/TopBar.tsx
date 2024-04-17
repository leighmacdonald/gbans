import { JSX, useMemo, useState, MouseEvent } from 'react';
import { Link as RouterLink } from 'react-router-dom';
import AccountCircleIcon from '@mui/icons-material/AccountCircle';
import ArticleIcon from '@mui/icons-material/Article';
import DarkModeIcon from '@mui/icons-material/DarkMode';
import DashboardIcon from '@mui/icons-material/Dashboard';
import DnsIcon from '@mui/icons-material/Dns';
import EmojiEventsIcon from '@mui/icons-material/EmojiEvents';
import ExitToAppIcon from '@mui/icons-material/ExitToApp';
import ForumIcon from '@mui/icons-material/Forum';
import GroupsIcon from '@mui/icons-material/Groups';
import HowToVoteIcon from '@mui/icons-material/HowToVote';
import LightModeIcon from '@mui/icons-material/LightMode';
import LiveHelpIcon from '@mui/icons-material/LiveHelp';
import MailIcon from '@mui/icons-material/Mail';
import MenuIcon from '@mui/icons-material/Menu';
import NewspaperIcon from '@mui/icons-material/Newspaper';
import NoAccountsIcon from '@mui/icons-material/NoAccounts';
import PersonSearchIcon from '@mui/icons-material/PersonSearch';
import PublicOffIcon from '@mui/icons-material/PublicOff';
import ReportIcon from '@mui/icons-material/Report';
import SettingsIcon from '@mui/icons-material/Settings';
import StorageIcon from '@mui/icons-material/Storage';
import SubjectIcon from '@mui/icons-material/Subject';
import SupportIcon from '@mui/icons-material/Support';
import TimelineIcon from '@mui/icons-material/Timeline';
import TravelExploreIcon from '@mui/icons-material/TravelExplore';
import WifiOffIcon from '@mui/icons-material/WifiOff';
import AppBar from '@mui/material/AppBar';
import Avatar from '@mui/material/Avatar';
import Badge from '@mui/material/Badge';
import Box from '@mui/material/Box';
import Button from '@mui/material/Button';
import Container from '@mui/material/Container';
import IconButton from '@mui/material/IconButton';
import Link from '@mui/material/Link';
import ListItemIcon from '@mui/material/ListItemIcon';
import ListItemText from '@mui/material/ListItemText';
import Menu from '@mui/material/Menu';
import MenuItem from '@mui/material/MenuItem';
import Toolbar from '@mui/material/Toolbar';
import Tooltip from '@mui/material/Tooltip';
import Typography from '@mui/material/Typography';
import { useTheme } from '@mui/material/styles';
import SteamID from 'steamid';
import { generateOIDCLink, PermissionLevel, UserNotification } from '../api';
import { NotificationsProvider } from '../contexts/NotificationsCtx';
import { useColourModeCtx } from '../hooks/useColourModeCtx.ts';
import { useCurrentUserCtx } from '../hooks/useCurrentUserCtx.ts';
import { useNotificationsCtx } from '../hooks/useNotificationsCtx.ts';
import steamLogo from '../icons/steam_login_sm.png';
import { tf2Fonts } from '../theme';
import { logErr } from '../util/errors';

interface menuRoute {
    to: string;
    text: string;
    icon: JSX.Element;
}

export const TopBar = () => {
    const { currentUser } = useCurrentUserCtx();
    const { notifications } = useNotificationsCtx();
    const theme = useTheme();
    const colourMode = useColourModeCtx();

    const [anchorElNav, setAnchorElNav] = useState<null | HTMLElement>(null);

    const [anchorElUser, setAnchorElUser] = useState<null | HTMLElement>(null);

    const [anchorElAdmin, setAnchorElAdmin] = useState<null | HTMLElement>(
        null
    );

    const handleOpenNavMenu = (event: MouseEvent<HTMLElement>) => {
        setAnchorElNav(event.currentTarget);
    };

    const handleOpenUserMenu = (event: MouseEvent<HTMLElement>) => {
        setAnchorElUser(event.currentTarget);
    };

    const handleOpenAdminMenu = (event: MouseEvent<HTMLElement>) => {
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
        ];
        if (currentUser.ban_id <= 0) {
            items.push({
                to: '/servers',
                text: 'Servers',
                icon: <StorageIcon sx={topColourOpts} />
            });
        }
        if (currentUser.permission_level >= PermissionLevel.Moderator) {
            items.push({
                to: '/forums',
                text: 'Forums',
                icon: <ForumIcon sx={topColourOpts} />
            });
        }
        items.push({
            to: '/wiki',
            text: 'Wiki',
            icon: <ArticleIcon sx={topColourOpts} />
        });
        if (currentUser.ban_id <= 0) {
            items.push({
                to: '/report',
                text: 'Report',
                icon: <ReportIcon sx={topColourOpts} />
            });
        }
        if (currentUser.ban_id > 0) {
            items.push({
                to: `/ban/${currentUser.ban_id}`,
                text: 'Appeal',
                icon: <SupportIcon sx={topColourOpts} />
            });
        }
        return items;
    }, [currentUser.ban_id, currentUser.permission_level, topColourOpts]);

    const userItems: menuRoute[] = useMemo(
        () => [
            {
                to: `/profile/${currentUser?.steam_id}`,
                text: 'Profile',
                icon: <AccountCircleIcon sx={colourOpts} />
            },
            {
                to: '/settings',
                text: 'Settings',
                icon: <SettingsIcon sx={colourOpts} />
            },

            {
                to: `/logs/${currentUser?.steam_id}`,
                text: 'Match History',
                icon: <TimelineIcon sx={colourOpts} />
            },
            {
                to: '/logout',
                text: 'Logout',
                icon: <ExitToAppIcon sx={colourOpts} />
            }
        ],
        [colourOpts, currentUser?.steam_id]
    );

    const adminItems: menuRoute[] = useMemo(() => {
        const items: menuRoute[] = [];
        if (currentUser.permission_level >= PermissionLevel.Editor) {
            items.push({
                to: '/admin/filters',
                text: 'Filtered Words',
                icon: <SubjectIcon sx={colourOpts} />
            });
        }
        if (currentUser.permission_level >= PermissionLevel.Moderator) {
            items.push({
                to: '/admin/ban/steam',
                text: 'Ban Steam',
                icon: <NoAccountsIcon sx={colourOpts} />
            });
            items.push({
                to: '/admin/ban/cidr',
                text: 'Ban CIDR',
                icon: <WifiOffIcon sx={colourOpts} />
            });
            items.push({
                to: '/admin/ban/asn',
                text: 'Ban ASN',
                icon: <PublicOffIcon sx={colourOpts} />
            });
            items.push({
                to: '/admin/ban/group',
                text: 'Ban Steam Group',
                icon: <GroupsIcon sx={colourOpts} />
            });
            items.push({
                to: '/admin/reports',
                text: 'Reports',
                icon: <ReportIcon sx={colourOpts} />
            });
            items.push({
                to: '/admin/appeals',
                text: 'Ban Appeals',
                icon: <LiveHelpIcon sx={colourOpts} />
            });
            items.push({
                to: '/admin/news',
                text: 'News',
                icon: <NewspaperIcon sx={colourOpts} />
            });
            items.push({
                to: '/admin/network',
                text: 'IP/Network Tools',
                icon: <TravelExploreIcon sx={colourOpts} />
            });
            items.push({
                to: '/admin/contests',
                text: 'Contests',
                icon: <EmojiEventsIcon sx={colourOpts} />
            });
            items.push({
                to: '/admin/people',
                text: 'People',
                icon: <PersonSearchIcon sx={colourOpts} />
            });
            items.push({
                to: '/admin/votes',
                text: 'Vote History',
                icon: <HowToVoteIcon sx={colourOpts} />
            });
        }
        if (currentUser.permission_level >= PermissionLevel.Admin) {
            // items.push({
            //     to: '/admin/people',
            //     text: 'People',
            //     icon: <PregnantWomanIcon sx={colourOpts} />
            // });
            // items.push({
            //     to: '/admin/import',
            //     text: 'Import',
            //     icon: <ImportExportIcon sx={colourOpts} />
            // });
            items.push({
                to: '/admin/servers',
                text: 'Servers',
                icon: <DnsIcon sx={colourOpts} />
            });
        }
        return items;
    }, [colourOpts, currentUser.permission_level]);

    const renderLinkedMenuItem = (
        text: string,
        route: string,
        icon: JSX.Element
    ) => (
        <MenuItem
            component={RouterLink}
            to={route}
            onClick={() => {
                setAnchorElNav(null);
                setAnchorElUser(null);
                setAnchorElAdmin(null);
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

    const validSteamId = useMemo(() => {
        try {
            if (currentUser?.steam_id) {
                const sid = new SteamID(currentUser?.steam_id);
                return sid.isValidIndividual();
            }
            return false;
        } catch (e) {
            logErr(e);
        }
        return false;
    }, [currentUser?.steam_id]);

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
                        {window.gbans.site_name || 'gbans'}
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
                        {window.gbans.site_name || 'gbans'}
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
                            {currentUser.permission_level >=
                                PermissionLevel.Admin && (
                                <NotificationsProvider>
                                    <IconButton
                                        component={RouterLink}
                                        to={'/notifications'}
                                        color={'inherit'}
                                    >
                                        <Badge
                                            badgeContent={
                                                (notifications ?? []).filter(
                                                    (n: UserNotification) =>
                                                        !n.read
                                                ).length
                                            }
                                        >
                                            <MailIcon />
                                        </Badge>
                                    </IconButton>
                                </NotificationsProvider>
                            )}
                            {!currentUser ||
                                (!validSteamId && (
                                    <Tooltip title="Steam Login">
                                        <Button
                                            component={Link}
                                            href={generateOIDCLink(
                                                window.location.pathname
                                            )}
                                        >
                                            <img
                                                src={steamLogo}
                                                alt={'Steam Login'}
                                            />
                                        </Button>
                                    </Tooltip>
                                ))}
                            {currentUser &&
                                validSteamId &&
                                adminItems.length > 0 && (
                                    <>
                                        <Tooltip title="Mod/Admin">
                                            <IconButton
                                                color={'inherit'}
                                                onClick={handleOpenAdminMenu}
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

                            {currentUser && validSteamId && (
                                <>
                                    <Tooltip title="User Settings">
                                        <IconButton
                                            onClick={handleOpenUserMenu}
                                            sx={{ p: 0 }}
                                        >
                                            <Avatar
                                                alt={currentUser.name}
                                                src={currentUser.avatarhash}
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
        </AppBar>
    );
};
