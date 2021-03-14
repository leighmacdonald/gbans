import React from 'react';
import {AppBar, Box, Grid, Tab, Tabs, Typography} from '@material-ui/core';
import {PlayerProfile} from '../util/api';

interface TabPanelProps {
    children?: React.ReactNode;
    index: any;
    value: any;
}

function TabPanel(props: TabPanelProps) {
    const {children, value, index, ...other} = props;

    return (
        <div role="tabpanel" hidden={value !== index} id={`wrapped-tabpanel-${index}`} aria-labelledby={`wrapped-tab-${index}`} {...other}>
            {value === index && (
                <Box p={3}>
                    <Typography>{children}</Typography>
                </Box>
            )}
        </div>
    );
}

export interface ProfilePanelProps {
    profile?: PlayerProfile;
}

export const a11yProps = (index: any) => {
    return {
        id: `wrapped-tab-${index}`,
        'aria-controls': `wrapped-tabpanel-${index}`
    };
};

export const ProfilePanel = (args: ProfilePanelProps) => {
    // const [friendsPage, setFriendsPage] = React.useState<number>(0);
    // const [showFriends, setShowFriends] = React.useState<boolean>(false);

    const [value, setValue] = React.useState('one');

    const handleChange = (_: React.ChangeEvent<{}>, newValue: string) => {
        setValue(newValue);
    };

    return (
        <Grid container>
            <Grid item>
                <AppBar position="static">
                    <Tabs value={value} onChange={handleChange} aria-label="wrapped label tabs example">
                        <Tab value="one" label="Profile" wrapped {...a11yProps('Profile')} />
                        <Tab value="two" label="Friends" {...a11yProps('Friends')} />
                    </Tabs>
                </AppBar>
                <TabPanel value={value} index="one">
                    <Grid container>
                        <Grid item>
                            <img src={args.profile?.player.avatarfull} alt={'Avatar'} />
                        </Grid>
                        <Grid item>
                            <Typography variant={'h3'}>{args.profile?.player.personaname}</Typography>
                        </Grid>
                    </Grid>
                </TabPanel>
                <TabPanel value={value} index="two">
                    Friends List...
                </TabPanel>
            </Grid>
        </Grid>
    );
};
