import { JSX, MouseEvent, useMemo, useState } from 'react';
import AccountCircleIcon from '@mui/icons-material/AccountCircle';
import AddModeratorIcon from '@mui/icons-material/AddModerator';
import ArticleIcon from '@mui/icons-material/Article';
import BlockIcon from '@mui/icons-material/Block';
import CellTowerIcon from '@mui/icons-material/CellTower';
import DarkModeIcon from '@mui/icons-material/DarkMode';
import DashboardIcon from '@mui/icons-material/Dashboard';
import DeveloperBoardIcon from '@mui/icons-material/DeveloperBoard';
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
import SensorOccupiedIcon from '@mui/icons-material/SensorOccupied';
import SettingsIcon from '@mui/icons-material/Settings';
import StorageIcon from '@mui/icons-material/Storage';
import SubjectIcon from '@mui/icons-material/Subject';
import SupportIcon from '@mui/icons-material/Support';
import TimelineIcon from '@mui/icons-material/Timeline';
import TravelExploreIcon from '@mui/icons-material/TravelExplore';
import WifiFindIcon from '@mui/icons-material/WifiFind';
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
import Stack from '@mui/material/Stack';
import Toolbar from '@mui/material/Toolbar';
import Tooltip from '@mui/material/Tooltip';
import Typography from '@mui/material/Typography';
import { useTheme } from '@mui/material/styles';
import { useQuery } from '@tanstack/react-query';
import { useNavigate } from '@tanstack/react-router';
import { MenuItemData, NestedDropdown } from 'mui-nested-menu';
import { apiGetNotifications, PermissionLevel, UserNotification } from '../api';
import { useAppInfoCtx } from '../contexts/AppInfoCtx.ts';
import { useAuth } from '../hooks/useAuth.ts';
import { useColourModeCtx } from '../hooks/useColourModeCtx.ts';
import steamLogo from '../icons/steam_login_sm.png';
import { tf2Fonts } from '../theme';
import { generateOIDCLink } from '../util/auth/generateOIDCLink.ts';
import { DesktopNotifications } from './DesktopNotifications.tsx';
import { NotificationsProvider } from './NotificationsProvider.tsx';
import RouterLink from './RouterLink.tsx';
import { VCenterBox } from './VCenterBox.tsx';

interface menuRoute {
    to: string;
    text: string;
    icon: JSX.Element;
}

export const TopBar = () => {
    const { profile, hasPermission, isAuthenticated } = useAuth();
    const { appInfo } = useAppInfoCtx();

    const { data: notifications, isLoading } = useQuery({
        queryKey: ['notifications'],
        queryFn: async () => {
            if (profile.steam_id == '') {
                return [];
            }
            return (await apiGetNotifications()) ?? [];
        },
        refetchInterval: 60 * 1000,
        refetchIntervalInBackground: true,
        refetchOnWindowFocus: true
    });

    const theme = useTheme();
    const colourMode = useColourModeCtx();
    const navigate = useNavigate();

    const [anchorElNav, setAnchorElNav] = useState<null | HTMLElement>(null);

    const [anchorElUser, setAnchorElUser] = useState<null | HTMLElement>(null);

    const handleOpenNavMenu = (event: MouseEvent<HTMLElement>) => {
        setAnchorElNav(event.currentTarget);
    };

    const handleOpenUserMenu = (event: MouseEvent<HTMLElement>) => {
        setAnchorElUser(event.currentTarget);
    };

    const handleCloseNavMenu = () => {
        setAnchorElNav(null);
    };

    const handleCloseUserMenu = () => {
        setAnchorElUser(null);
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
        if (appInfo.servers_enabled && profile.ban_id <= 0) {
            items.push({
                to: '/servers',
                text: 'Servers',
                icon: <StorageIcon sx={topColourOpts} />
            });
        }
        if (appInfo.forums_enabled && hasPermission(PermissionLevel.Moderator)) {
            items.push({
                to: '/forums',
                text: 'Forums',
                icon: <ForumIcon sx={topColourOpts} />
            });
        }
        if (appInfo.wiki_enabled) {
            items.push({
                to: '/wiki',
                text: 'Wiki',
                icon: <ArticleIcon sx={topColourOpts} />
            });
        }
        if (appInfo.reports_enabled) {
            if (profile.ban_id <= 0) {
                items.push({
                    to: '/report',
                    text: 'Report',
                    icon: <ReportIcon sx={topColourOpts} />
                });
            }
            if (profile.ban_id > 0) {
                items.push({
                    to: `/ban/${profile.ban_id}`,
                    text: 'Appeal',
                    icon: <SupportIcon sx={topColourOpts} />
                });
            }
        }
        return items;
    }, [
        appInfo.forums_enabled,
        appInfo.reports_enabled,
        appInfo.servers_enabled,
        appInfo.wiki_enabled,
        hasPermission,
        profile.ban_id,
        topColourOpts
    ]);

    const userItems: menuRoute[] = useMemo(() => {
        const items = [
            {
                to: `/profile/${profile.steam_id}`,
                text: 'Profile',
                icon: <AccountCircleIcon sx={colourOpts} />
            },
            {
                to: '/settings',
                text: 'Settings',
                icon: <SettingsIcon sx={colourOpts} />
            }
        ];
        if (appInfo.stats_enabled) {
            items.push({
                to: `/logs/${profile.steam_id}`,
                text: 'Match History',
                icon: <TimelineIcon sx={colourOpts} />
            });
        }
        items.push({
            to: '/logout',
            text: 'Logout',
            icon: <ExitToAppIcon sx={colourOpts} />
        });
        return items;
    }, [appInfo.stats_enabled, colourOpts, profile.steam_id]);

    // @ts-expect-error label defined as string
    const adminItems: MenuItemData = useMemo(() => {
        const onClickHandler = (href: string) => {
            return async () => {
                await navigate({ to: href });
            };
        };

        return {
            label: <SettingsIcon />,
            items: [
                {
                    leftIcon: <SubjectIcon sx={colourOpts} />,
                    label: 'Filtered Words',
                    callback: onClickHandler('/admin/filters')
                },
                {
                    leftIcon: <BlockIcon sx={colourOpts} />,
                    label: 'Ban',
                    items: [
                        {
                            leftIcon: <NoAccountsIcon sx={colourOpts} />,
                            label: 'Steam',
                            callback: onClickHandler('/admin/ban/steam')
                        },
                        {
                            leftIcon: <WifiOffIcon sx={colourOpts} />,
                            label: 'IP/CIDR',
                            callback: onClickHandler('/admin/ban/cidr')
                        },
                        {
                            leftIcon: <GroupsIcon sx={colourOpts} />,
                            label: 'Steam Group',
                            callback: onClickHandler('/admin/ban/group')
                        },
                        {
                            leftIcon: <PublicOffIcon sx={colourOpts} />,
                            label: 'ASN',
                            callback: onClickHandler('/admin/ban/asn')
                        }
                    ]
                },

                {
                    leftIcon: <ReportIcon sx={colourOpts} />,
                    label: 'Reports',
                    callback: onClickHandler('/admin/reports')
                },
                {
                    leftIcon: <LiveHelpIcon sx={colourOpts} />,
                    label: 'Ban Appeals',
                    callback: onClickHandler('/admin/appeals')
                },
                {
                    label: 'News',
                    leftIcon: <NewspaperIcon sx={colourOpts} />,
                    callback: onClickHandler('/admin/news')
                },
                {
                    leftIcon: <TravelExploreIcon sx={colourOpts} />,
                    label: 'IP/Network Tools',
                    callback: onClickHandler('/admin/network'),
                    items: [
                        {
                            leftIcon: <SensorOccupiedIcon sx={colourOpts} />,
                            label: 'Player IP History',
                            callback: onClickHandler('/admin/network/iphist')
                        },
                        {
                            leftIcon: <WifiFindIcon sx={colourOpts} />,
                            label: 'Find Players By IP',
                            callback: onClickHandler('/admin/network/playersbyip')
                        },
                        {
                            leftIcon: <CellTowerIcon sx={colourOpts} />,
                            label: 'IP Info',
                            callback: onClickHandler('/admin/network/ipinfo')
                        },
                        {
                            leftIcon: <WifiOffIcon sx={colourOpts} />,
                            label: 'External CIDR Bans',
                            callback: onClickHandler('/admin/network/cidrblocks')
                        }
                    ]
                },
                {
                    leftIcon: <EmojiEventsIcon sx={colourOpts} />,
                    label: 'Contests',
                    callback: onClickHandler('/admin/contests')
                },
                {
                    leftIcon: <PersonSearchIcon sx={colourOpts} />,
                    label: 'People',
                    callback: onClickHandler('/admin/people')
                },
                {
                    leftIcon: <HowToVoteIcon sx={colourOpts} />,
                    label: 'Vote History',
                    callback: onClickHandler('/admin/votes')
                },
                {
                    leftIcon: <SettingsIcon sx={colourOpts} />,
                    label: 'Servers',
                    callback: onClickHandler('/admin/servers')
                },
                {
                    leftIcon: <AddModeratorIcon sx={colourOpts} />,
                    label: 'Game Admins',
                    callback: onClickHandler('/admin/game-admins')
                },
                {
                    leftIcon: <DeveloperBoardIcon sx={colourOpts} />,
                    label: 'System Settings',
                    callback: onClickHandler('/admin/settings')
                }
            ]
        };
    }, [colourOpts, navigate]);

    const renderLinkedMenuItem = (text: string, route: string, icon: JSX.Element) => (
        <MenuItem
            component={RouterLink}
            to={route}
            onClick={() => {
                setAnchorElNav(null);
                setAnchorElUser(null);
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
                        {appInfo.site_name}
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
                                return renderLinkedMenuItem(value.text, value.to, value.icon);
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
                        {appInfo.site_name}
                    </Typography>
                    <Box
                        sx={{
                            flexGrow: 1,
                            display: { xs: 'none', md: 'flex' }
                        }}
                    >
                        {menuItems.map((value) => {
                            return renderLinkedMenuItem(value.text, value.to, value.icon);
                        })}
                    </Box>

                    <Box sx={{ flexGrow: 0 }}>
                        <Stack direction={'row'} spacing={1}>
                            <Tooltip title="Toggle BLU/RED mode">
                                <IconButton onClick={colourMode.toggleColorMode}>{themeIcon}</IconButton>
                            </Tooltip>

                            {hasPermission(PermissionLevel.User) && (
                                <NotificationsProvider>
                                    <IconButton component={RouterLink} to={'/notifications'} color={'inherit'}>
                                        <Badge
                                            color={'success'}
                                            badgeContent={
                                                isLoading
                                                    ? '...'
                                                    : (notifications ?? []).filter((n: UserNotification) => !n.read)
                                                          .length
                                            }
                                        >
                                            <MailIcon />
                                        </Badge>
                                    </IconButton>
                                </NotificationsProvider>
                            )}

                            {!isAuthenticated() && (
                                <Tooltip title="Steam Login">
                                    <Button component={Link} href={generateOIDCLink(window.location.pathname)}>
                                        <img src={steamLogo} alt={'Steam Login'} />
                                    </Button>
                                </Tooltip>
                            )}
                            {hasPermission(PermissionLevel.Moderator) && (
                                <VCenterBox>
                                    <NestedDropdown
                                        menuItemsData={adminItems}
                                        MenuProps={{
                                            elevation: 0
                                        }}
                                        ButtonProps={{
                                            sx: { color: 'common.white' },
                                            variant: 'text'
                                        }}
                                    />
                                </VCenterBox>
                            )}

                            {isAuthenticated() && (
                                <>
                                    <Tooltip title="User Settings">
                                        <IconButton onClick={handleOpenUserMenu} sx={{ p: 0 }}>
                                            <Avatar alt={profile.name} src={profile.avatarhash} />
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
                                            return renderLinkedMenuItem(value.text, value.to, value.icon);
                                        })}
                                    </Menu>
                                </>
                            )}
                        </Stack>
                    </Box>
                </Toolbar>
            </Container>
            <DesktopNotifications notifications={notifications} isLoading={isLoading} />
        </AppBar>
    );
};
