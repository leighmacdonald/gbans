import React from 'react';
import {
    AppBar,
    Chip,
    Grid,
    Paper,
    Tab,
    Tabs,
    Typography
} from '@material-ui/core';
import { PlayerProfile } from '../util/api';
import { makeStyles, Theme } from '@material-ui/core/styles';
import CheckIcon from '@material-ui/icons/Check';
import ClearIcon from '@material-ui/icons/Clear';
import { GLink } from './GLink';

interface TabPanelProps {
    children?: React.ReactNode;
    index: number | string;
    value: number | string;
}

const useStyles = makeStyles((theme: Theme) => ({
    paper: {
        padding: theme.spacing(2),
        textAlign: 'center',
        color: theme.palette.text.secondary
    },
    ok: {
        backgroundColor: theme.palette.success.main
    },
    error: {
        backgroundColor: theme.palette.error.main
    }
}));

function TabPanel(props: TabPanelProps) {
    const classes = useStyles();
    const { children, value, index, ...other } = props;

    return (
        <Paper
            className={classes.paper}
            variant={'outlined'}
            role="tabpanel"
            hidden={value !== index}
            id={`wrapped-tabpanel-${index}`}
            aria-labelledby={`wrapped-tab-${index}`}
            {...other}
        >
            {value === index && <>{children}</>}
        </Paper>
    );
}

export interface ProfilePanelProps {
    profile?: PlayerProfile;
}

export const a11yProps = (index: number | string): Record<string, string> => {
    return {
        id: `wrapped-tab-${index}`,
        'aria-controls': `wrapped-tabpanel-${index}`
    };
};

export const ProfilePanel = (args: ProfilePanelProps): JSX.Element => {
    const classes = useStyles();
    const [value, setValue] = React.useState('one');

    const handleChange = (
        _: React.ChangeEvent<Record<string, unknown>>,
        newValue: string
    ) => {
        setValue(newValue);
    };

    return (
        <Grid container>
            {!args.profile && (
                <Grid item xs={12}>
                    <Typography variant={'h3'}>No Profile Selected</Typography>
                </Grid>
            )}
            {args.profile && (
                <Grid item xs={12}>
                    <AppBar position="static">
                        <Tabs
                            value={value}
                            onChange={handleChange}
                            aria-label="Player Profile Panel"
                            variant={'fullWidth'}
                        >
                            <Tab
                                value="one"
                                label="Profile"
                                {...a11yProps('Profile')}
                            />
                            <Tab
                                value="two"
                                label={
                                    'Friends' +
                                    (args.profile?.friends
                                        ? ` (${args.profile?.friends.length})`
                                        : '')
                                }
                                {...a11yProps('Friends')}
                            />
                        </Tabs>
                    </AppBar>
                    <TabPanel value={value} index="one">
                        <Grid container>
                            <Grid item xs>
                                <img
                                    src={args.profile?.player.avatarfull}
                                    alt={'Avatar'}
                                />
                            </Grid>
                            <Grid item xs>
                                <Typography variant={'h3'} align={'center'}>
                                    {args.profile?.player.personaname}
                                </Typography>
                            </Grid>
                            <Grid container>
                                <Grid item xs={3}>
                                    <Chip
                                        className={classes.ok}
                                        label={'VAC'}
                                        icon={<CheckIcon />}
                                    />
                                </Grid>
                                <Grid item xs={3}>
                                    <Chip
                                        className={classes.ok}
                                        label={'Trade'}
                                        icon={<CheckIcon />}
                                    />
                                </Grid>
                                <Grid item xs={3}>
                                    <Chip
                                        className={classes.ok}
                                        label="Community"
                                        icon={<CheckIcon />}
                                    />
                                </Grid>
                                <Grid item xs={3}>
                                    <Chip
                                        className={classes.error}
                                        label={'Game'}
                                        icon={<ClearIcon />}
                                    />
                                </Grid>
                            </Grid>
                        </Grid>
                    </TabPanel>
                    <TabPanel value={value} index="two">
                        <Grid container>
                            {args.profile.friends?.map((p) => (
                                <Grid container key={p.steamid}>
                                    <Grid item xs={3}>
                                        <img
                                            src={p.avatar}
                                            alt={'Profile Picture'}
                                        />
                                    </Grid>
                                    <Grid item xs={9}>
                                        <GLink
                                            to={`https://steamcommunity.com/profiles/${p.steam_id}`}
                                            primary={p.personaname}
                                        />
                                    </Grid>
                                </Grid>
                            ))}
                        </Grid>
                    </TabPanel>
                </Grid>
            )}
        </Grid>
    );
};
