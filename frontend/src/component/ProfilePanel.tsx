import AppBar from '@mui/material/AppBar';
import Avatar from '@mui/material/Avatar';
import Chip from '@mui/material/Chip';
import List from '@mui/material/List';
import ListItemAvatar from '@mui/material/ListItemAvatar';
import ListItemButton from '@mui/material/ListItemButton';
import ListItemText from '@mui/material/ListItemText';
import Tab from '@mui/material/Tab';
import Tabs from '@mui/material/Tabs';
import Typography from '@mui/material/Typography';
import Paper from '@mui/material/Paper';
import React from 'react';
import CheckIcon from '@mui/icons-material/Check';
import ClearIcon from '@mui/icons-material/Clear';
import { PlayerProfile } from '../api';
import Stack from '@mui/material/Stack';
import Box from '@mui/material/Box';

interface TabPanelProps {
    children?: React.ReactNode;
    index: number | string;
    value: number | string;
}

function TabPanel(props: TabPanelProps) {
    const { children, value, index, ...other } = props;

    return (
        <Paper
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
    const [value, setValue] = React.useState('one');

    const handleChange = (
        _: React.ChangeEvent<Record<string, unknown>> | any,
        newValue: string
    ) => {
        setValue(newValue);
    };

    return (
        <Stack spacing={3} padding={3}>
            {!args.profile && (
                <Box>
                    <Typography variant={'h3'}>No Profile Selected</Typography>
                </Box>
            )}
            {args.profile && (
                <Box>
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
                        <Stack>
                            <img
                                src={args.profile?.player.avatarfull}
                                alt={'Avatar'}
                            />
                            <Typography variant={'h3'} align={'center'}>
                                {args.profile?.player.personaname}
                            </Typography>
                            <Stack direction={'row'}>
                                <Chip label={'VAC'} icon={<CheckIcon />} />
                                <Chip label={'Trade'} icon={<CheckIcon />} />
                                <Chip label="Community" icon={<CheckIcon />} />
                                <Chip label={'Game'} icon={<ClearIcon />} />
                            </Stack>
                        </Stack>
                    </TabPanel>
                    <TabPanel value={value} index="two">
                        <List dense={false}>
                            {args.profile.friends?.map((p) => (
                                <ListItemButton key={p.steamid}>
                                    <ListItemAvatar>
                                        <Avatar
                                            alt={'Profile Picture'}
                                            src={p.avatar}
                                        />
                                    </ListItemAvatar>
                                    <ListItemText
                                        primary={p.personaname}
                                        secondary={p.steamid}
                                    />
                                </ListItemButton>
                            ))}
                        </List>
                    </TabPanel>
                </Box>
            )}
        </Stack>
    );
};
