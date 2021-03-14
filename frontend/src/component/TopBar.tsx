import React, {useState} from 'react';
import {fade, makeStyles, Theme, useTheme} from '@material-ui/core/styles';
import AppBar from '@material-ui/core/AppBar';
import Toolbar from '@material-ui/core/Toolbar';
import IconButton from '@material-ui/core/IconButton';
import Typography from '@material-ui/core/Typography';
import InputBase from '@material-ui/core/InputBase';
import Badge from '@material-ui/core/Badge';
import MenuItem from '@material-ui/core/MenuItem';
import Menu from '@material-ui/core/Menu';
import MenuIcon from '@material-ui/icons/Menu';
import SearchIcon from '@material-ui/icons/Search';
import AccountCircle from '@material-ui/icons/AccountCircle';
import MailIcon from '@material-ui/icons/Mail';
import NotificationsIcon from '@material-ui/icons/Notifications';
import MoreIcon from '@material-ui/icons/MoreVert';
import {useCurrentUserCtx} from "../contexts/CurrentUserCtx";
import {handleOnLogin} from "../util/api";
import {Avatar, Divider} from "@material-ui/core";
import SettingsIcon from '@material-ui/icons/Settings';
import clsx from "clsx";
import ChevronRightIcon from "@material-ui/icons/ChevronRight";
import ChevronLeftIcon from "@material-ui/icons/ChevronLeft";
import List from "@material-ui/core/List";
import ListItem from "@material-ui/core/ListItem";
import ListItemIcon from "@material-ui/core/ListItemIcon";
import InboxIcon from "@material-ui/icons/MoveToInbox";
import ListItemText from "@material-ui/core/ListItemText";
import Drawer from "@material-ui/core/Drawer";
import {GLink} from "./GLink";

const drawerWidth = 240;

const useStyles = makeStyles((theme: Theme) => ({
    grow: {
        flexGrow: 1,
    },
    menuButton: {
        marginRight: theme.spacing(2),
    },
    title: {
        display: 'none',
        [theme.breakpoints.up('sm')]: {
            display: 'block',
        },
    },
    search: {
        position: 'relative',
        borderRadius: theme.shape.borderRadius,
        backgroundColor: fade(theme.palette.common.white, 0.15),
        '&:hover': {
            backgroundColor: fade(theme.palette.common.white, 0.25),
        },
        marginRight: theme.spacing(2),
        marginLeft: 0,
        width: '100%',
        [theme.breakpoints.up('sm')]: {
            marginLeft: theme.spacing(3),
            width: 'auto',
        },
    },
    searchIcon: {
        padding: theme.spacing(0, 2),
        height: '100%',
        position: 'absolute',
        pointerEvents: 'none',
        display: 'flex',
        alignItems: 'center',
        justifyContent: 'center',
    },
    inputRoot: {
        color: 'inherit',
    },
    inputInput: {
        padding: theme.spacing(1, 1, 1, 0),
        // vertical padding + font size from searchIcon
        paddingLeft: `calc(1em + ${theme.spacing(4)}px)`,
        transition: theme.transitions.create('width'),
        width: '100%',
        [theme.breakpoints.up('md')]: {
            width: '20ch',
        },
    },
    sectionDesktop: {
        display: 'none',
        [theme.breakpoints.up('md')]: {
            display: 'flex',
        },
    },
    sectionMobile: {
        display: 'flex',
        [theme.breakpoints.up('md')]: {
            display: 'none',
        },
    },
    root: {
        display: 'flex',
    },
    appBar: {
        zIndex: theme.zIndex.drawer + 1,
        transition: theme.transitions.create(['width', 'margin'], {
            easing: theme.transitions.easing.sharp,
            duration: theme.transitions.duration.leavingScreen,
        }),
    },
    appBarShift: {
        marginLeft: drawerWidth,
        width: `calc(100% - ${drawerWidth}px)`,
        transition: theme.transitions.create(['width', 'margin'], {
            easing: theme.transitions.easing.sharp,
            duration: theme.transitions.duration.enteringScreen,
        }),
    },
    hide: {
        display: 'none',
    },
    drawer: {
        width: drawerWidth,
        flexShrink: 0,
        whiteSpace: 'nowrap',
    },
    drawerOpen: {
        width: drawerWidth,
        transition: theme.transitions.create('width', {
            easing: theme.transitions.easing.sharp,
            duration: theme.transitions.duration.enteringScreen,
        }),
    },
    drawerClose: {
        transition: theme.transitions.create('width', {
            easing: theme.transitions.easing.sharp,
            duration: theme.transitions.duration.leavingScreen,
        }),
        overflowX: 'hidden',
        width: theme.spacing(7) + 1,
        [theme.breakpoints.up('sm')]: {
            width: theme.spacing(9) + 1,
        },
    },
    toolbar: {
        display: 'flex',
        alignItems: 'center',
        justifyContent: 'flex-end',
        padding: theme.spacing(0, 1),
        // necessary for content to be below app bar
        ...theme.mixins.toolbar,
    },
}));


export const TopBar = () => {
    const classes = useStyles();
    const [anchorProfileMenuEl, setAnchorProfileMenuEl] = useState(null);
    const [anchorAdminMenuEl, setAnchorAdminMenuEl] = useState(null);
    const [mobileMoreAnchorEl, setMobileMoreAnchorEl] = useState(null);

    const isProfileMenuOpen = Boolean(anchorProfileMenuEl);
    const isAdminMenuOpen = Boolean(anchorAdminMenuEl);
    const isMobileMenuOpen = Boolean(mobileMoreAnchorEl);
    const [sideMenuOpen, setSideMenuOpen] = useState<boolean>(false);
    const {currentUser} = useCurrentUserCtx();

    const handleAdminMenuOpen = (event: any) => {
        setAnchorAdminMenuEl(event.currentTarget);
    };
    const handleAdminMenuClose = () => {
        setAnchorAdminMenuEl(null);
        handleMobileMenuClose();
    };

    const handleProfileMenuOpen = (event: any) => {
        setAnchorProfileMenuEl(event.currentTarget);
    };

    const handleMobileMenuClose = () => {
        setMobileMoreAnchorEl(null);
    };

    const handleProfileMenuClose = () => {
        setAnchorProfileMenuEl(null);
        handleMobileMenuClose();
    };

    const handleMobileMenuOpen = (event: any) => {
        setMobileMoreAnchorEl(event.currentTarget);
    };

    const menuId = 'primary-search-account-menu';
    const adminMenuId = 'admin-menu';
    const handleSideMenuClose = () => {
        setSideMenuOpen(false)
    }
    const renderProfileMenu = (
        <Menu
            anchorEl={anchorProfileMenuEl}
            anchorOrigin={{vertical: 'top', horizontal: 'right'}}
            id={menuId}
            keepMounted
            transformOrigin={{vertical: 'top', horizontal: 'right'}}
            open={isProfileMenuOpen}
            onClose={handleProfileMenuClose}
        >
            <MenuItem onClick={handleProfileMenuClose}><GLink to={"/profile"} primary={"Profile"} /></MenuItem>
            <MenuItem onClick={handleProfileMenuClose}><GLink to={"/settings"} primary={"Settings"} /></MenuItem>
            <Divider light/>
            <MenuItem onClick={handleProfileMenuClose}><GLink to={"/logout"} primary={"Logout"} /></MenuItem>
        </Menu>
    );

    const renderAdminMenu = (
        <Menu
            anchorEl={anchorAdminMenuEl}
            anchorOrigin={{vertical: 'top', horizontal: 'right'}}
            id={adminMenuId}
            keepMounted
            transformOrigin={{vertical: 'top', horizontal: 'right'}}
            open={isAdminMenuOpen}
            onClose={handleAdminMenuClose}
        >
            <MenuItem onClick={handleAdminMenuClose}><GLink to={"/admin/ban"} primary={"Ban"} /></MenuItem>
            <MenuItem onClick={handleAdminMenuClose}><GLink to={"/admin/reports"} primary={"Reports"} /></MenuItem>
            <MenuItem onClick={handleAdminMenuClose}><GLink to={"/admin/people"} primary={"People"} /></MenuItem>
            <MenuItem onClick={handleAdminMenuClose}><GLink to={"/admin/import"} primary={"Import"} /></MenuItem>
            <MenuItem onClick={handleAdminMenuClose}><GLink to={"/admin/filters"} primary={"Filtered Words"} /></MenuItem>
            <MenuItem onClick={handleAdminMenuClose}><GLink to={"/admin/servers"} primary={"Servers"} /></MenuItem>
            <MenuItem onClick={handleAdminMenuClose}><GLink to={"/admin/success"} primary={"Server Logs"} /></MenuItem>
        </Menu>
    );

    const mobileMenuId = 'primary-search-account-menu-mobile';
    const renderMobileMenu = (
        <Menu
            anchorEl={mobileMoreAnchorEl}
            anchorOrigin={{vertical: 'top', horizontal: 'right'}}
            id={mobileMenuId}
            keepMounted
            transformOrigin={{vertical: 'top', horizontal: 'right'}}
            open={isMobileMenuOpen}
            onClose={handleMobileMenuClose}
        >
            <MenuItem>
                <IconButton aria-label="show 4 new mails" color="inherit">
                    <Badge badgeContent={4} color="secondary">
                        <MailIcon/>
                    </Badge>
                </IconButton>
                <p>Messages</p>
            </MenuItem>
            <MenuItem>
                <IconButton aria-label="show 11 new notifications" color="inherit">
                    <Badge badgeContent={11} color="secondary">
                        <NotificationsIcon/>
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
                    <AccountCircle/>
                </IconButton>
                <p>Profile</p>
            </MenuItem>
        </Menu>
    );
    const theme = useTheme();
    const renderSideMenu = (
        <Drawer
            variant="permanent"
            className={clsx(classes.drawer, {
                [classes.drawerOpen]: sideMenuOpen,
                [classes.drawerClose]: !sideMenuOpen,
            })}
            classes={{
                paper: clsx({
                    [classes.drawerOpen]: sideMenuOpen,
                    [classes.drawerClose]: !sideMenuOpen,
                }),
            }}
        >
            <div className={classes.toolbar}>
                <IconButton onClick={handleSideMenuClose}>
                    {theme.direction === 'rtl' ? <ChevronRightIcon/> : <ChevronLeftIcon/>}
                </IconButton>
            </div>
            <Divider/>
            <List>
                {['Inbox', 'Starred', 'Send email', 'Drafts'].map((text, index) => (
                    <ListItem button key={text}>
                        <ListItemIcon>{index % 2 === 0 ? <InboxIcon/> : <MailIcon/>}</ListItemIcon>
                        <ListItemText primary={text}/>
                    </ListItem>
                ))}
            </List>
            <Divider/>
            <List>
                {['All mail', 'Trash', 'Spam'].map((text, index) => (
                    <ListItem button key={text}>
                        <ListItemIcon>{index % 2 === 0 ? <InboxIcon/> : <MailIcon/>}</ListItemIcon>
                        <ListItemText primary={text}/>
                    </ListItem>
                ))}
            </List>
        </Drawer>
    )

    return (
        <>
            {renderSideMenu}

                <div className={classes.grow}>
                    <AppBar position="fixed" className={clsx(classes.appBar, {
                        [classes.appBarShift]: sideMenuOpen,
                    })}>
                        <Toolbar>
                            <IconButton
                                color="inherit"
                                aria-label="open drawer"
                                onClick={() => {
                                    setSideMenuOpen(true)
                                }}
                                edge="start"
                                className={clsx(classes.menuButton, {
                                    [classes.hide]: sideMenuOpen,
                                })}
                            >
                                <MenuIcon/>
                            </IconButton>
                            <Typography className={classes.title} variant="h6" noWrap>
                                gbans
                            </Typography>
                            <GLink to={"/bans"} primary={"Bans"}/>
                            <GLink to={"/settings"} primary={"Settings"}/>
                            <div className={classes.search}>
                                <div className={classes.searchIcon}>
                                    <SearchIcon/>
                                </div>
                                <InputBase
                                    placeholder="Search…"
                                    classes={{
                                        root: classes.inputRoot,
                                        input: classes.inputInput,
                                    }}
                                    inputProps={{'aria-label': 'search'}}
                                />
                            </div>
                            <div className={classes.grow}/>
                            <div className={classes.sectionDesktop}>
                                {currentUser?.player.steam_id <= 0 && <>
                                    <IconButton
                                        edge="end"
                                        aria-label="account of current user"
                                        aria-controls={menuId}
                                        aria-haspopup="true"
                                        onClick={handleOnLogin}
                                        color="inherit"
                                    >
                                    </IconButton>
                                </>}
                                {currentUser?.player.steam_id > 0 && <>
                                    <IconButton aria-label="show 4 new alerts" color="inherit">
                                        <Badge badgeContent={4} color="error">
                                            <MailIcon/>
                                        </Badge>
                                    </IconButton>
                                    <IconButton aria-label="show 17 new notifications" color="inherit">
                                        <Badge badgeContent={17} color="error">
                                            <NotificationsIcon/>
                                        </Badge>
                                    </IconButton>
                                    <IconButton
                                        edge="end"
                                        aria-label="admin menu"
                                        aria-controls={menuId}
                                        aria-haspopup="true"
                                        onClick={handleAdminMenuOpen}
                                        color="inherit"
                                    >
                                        <SettingsIcon/>
                                    </IconButton>
                                    <IconButton
                                        edge="end"
                                        aria-label="account of current user"
                                        aria-controls={menuId}
                                        aria-haspopup="true"
                                        onClick={handleProfileMenuOpen}
                                        color="inherit"
                                    >
                                        <Avatar alt={currentUser.player.personaname} src={currentUser.player.avatar}/>
                                    </IconButton>
                                </>
                                }
                            </div>
                            <div className={classes.sectionMobile}>
                                <IconButton
                                    aria-label="show more"
                                    aria-controls={mobileMenuId}
                                    aria-haspopup="true"
                                    onClick={handleMobileMenuOpen}
                                    color="inherit"
                                >
                                    <MoreIcon/>
                                </IconButton>
                            </div>
                        </Toolbar>
                    </AppBar>

                    {renderAdminMenu}
                    {renderMobileMenu}
                    {renderProfileMenu}

                </div>

        </>
    );
}