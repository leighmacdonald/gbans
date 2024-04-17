import { JSX, useMemo, useState, MouseEvent } from 'react';
import { Link as RouterLink } from 'react-router-dom';
import AccountCircleIcon from '@mui/icons-material/AccountCircle';
import ArticleIcon from '@mui/icons-material/Article';
import BlockIcon from '@mui/icons-material/Block';
import CellTowerIcon from '@mui/icons-material/CellTower';
import DarkModeIcon from '@mui/icons-material/DarkMode';
import DashboardIcon from '@mui/icons-material/Dashboard';
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
import SteamID from 'steamid';
import { generateOIDCLink, PermissionLevel, UserNotification } from '../api';
import { NotificationsProvider } from '../contexts/NotificationsCtx';
import { useColourModeCtx } from '../hooks/useColourModeCtx.ts';
import { useCurrentUserCtx } from '../hooks/useCurrentUserCtx.ts';
import { useNotificationsCtx } from '../hooks/useNotificationsCtx.ts';
import steamLogo from '../icons/steam_login_sm.png';
import { tf2Fonts } from '../theme';
import { logErr } from '../util/errors';
import { VCenterBox } from './VCenterBox.tsx';
import { MenuItemData, NestedDropdown } from './nested-menu';

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

    const adminItems: MenuItemData = useMemo(() => {
        return {
            label: <SettingsIcon />,
            items: [
                {
                    href: '/admin/filters',
                    label: (
                        <Stack direction={'row'} spacing={1}>
                            <SubjectIcon sx={colourOpts} />
                            <Typography>Filtered Words</Typography>
                        </Stack>
                    )
                },
                {
                    label: (
                        <Stack direction={'row'} spacing={1}>
                            <BlockIcon sx={colourOpts} />
                            <Typography>Ban</Typography>
                        </Stack>
                    ),
                    items: [
                        {
                            href: '/admin/ban/steam',
                            label: (
                                <Stack direction={'row'} spacing={1}>
                                    <NoAccountsIcon sx={colourOpts} />
                                    <Typography>Steam</Typography>
                                </Stack>
                            )
                        },
                        {
                            href: '/admin/ban/cidr',
                            label: (
                                <Stack direction={'row'} spacing={1}>
                                    <WifiOffIcon sx={colourOpts} />
                                    <Typography>IP/CIDR</Typography>
                                </Stack>
                            )
                        },
                        {
                            href: '/admin/ban/group',
                            label: (
                                <Stack direction={'row'} spacing={1}>
                                    <GroupsIcon sx={colourOpts} />
                                    <Typography>Steam Group</Typography>
                                </Stack>
                            )
                        },
                        {
                            href: '/admin/ban/asn',
                            label: (
                                <Stack direction={'row'} spacing={1}>
                                    <PublicOffIcon sx={colourOpts} />
                                    <Typography>ASN</Typography>
                                </Stack>
                            )
                        }
                    ]
                },

                {
                    href: '/admin/reports',
                    label: (
                        <Stack direction={'row'} spacing={1}>
                            <ReportIcon sx={colourOpts} />
                            <Typography>Reports</Typography>
                        </Stack>
                    )
                },
                {
                    href: '/admin/appeals',
                    label: (
                        <Stack direction={'row'} spacing={1}>
                            <LiveHelpIcon sx={colourOpts} />
                            <Typography>Ban Appeals</Typography>
                        </Stack>
                    )
                },
                {
                    href: '/admin/news',
                    label: (
                        <Stack direction={'row'} spacing={1}>
                            <NewspaperIcon sx={colourOpts} />
                            <Typography>News</Typography>
                        </Stack>
                    )
                },
                {
                    href: '/admin/network',
                    label: (
                        <Stack direction={'row'} spacing={1}>
                            <TravelExploreIcon sx={colourOpts} />
                            <Typography>IP/Network Tools</Typography>
                        </Stack>
                    ),
                    items: [
                        {
                            href: '/admin/network/ip_hist',
                            label: (
                                <Stack direction={'row'} spacing={1}>
                                    <SensorOccupiedIcon sx={colourOpts} />
                                    <Typography>Player IP History</Typography>
                                </Stack>
                            )
                        },
                        {
                            href: '/admin/network/players_by_ip',
                            label: (
                                <Stack direction={'row'} spacing={1}>
                                    <WifiFindIcon sx={colourOpts} />
                                    <Typography>Find Players By IP</Typography>
                                </Stack>
                            )
                        },
                        {
                            href: '/admin/network/ip_info',
                            label: (
                                <Stack direction={'row'} spacing={1}>
                                    <CellTowerIcon sx={colourOpts} />
                                    <Typography>IP Info</Typography>
                                </Stack>
                            )
                        },
                        {
                            href: '/admin/network/cidr_blocks',
                            label: (
                                <Stack direction={'row'} spacing={1}>
                                    <WifiOffIcon sx={colourOpts} />
                                    <Typography>External CIDR Bans</Typography>
                                </Stack>
                            )
                        }
                    ]
                },
                {
                    href: '/admin/contests',
                    label: (
                        <Stack direction={'row'} spacing={1}>
                            <EmojiEventsIcon sx={colourOpts} />
                            <Typography>Contests</Typography>
                        </Stack>
                    )
                },
                {
                    href: '/admin/people',
                    label: (
                        <Stack direction={'row'} spacing={1}>
                            <PersonSearchIcon sx={colourOpts} />
                            <Typography>People</Typography>
                        </Stack>
                    )
                },
                {
                    href: '/admin/votes',
                    label: (
                        <Stack direction={'row'} spacing={1}>
                            <HowToVoteIcon sx={colourOpts} />
                            <Typography>Vote History</Typography>
                        </Stack>
                    )
                },
                {
                    href: '/admin/servers',
                    label: (
                        <Stack direction={'row'} spacing={1}>
                            <SettingsIcon sx={colourOpts} />
                            <Typography>Servers</Typography>
                        </Stack>
                    )
                }
            ]
        };
    }, [colourOpts]);

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
                        <Stack direction={'row'} spacing={1}>
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
                                currentUser.permission_level >=
                                    PermissionLevel.Moderator && (
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
                        </Stack>
                    </Box>
                </Toolbar>
            </Container>
        </AppBar>
    );
};
