import { SyntheticEvent, useState } from 'react';
import HelpIcon from '@mui/icons-material/Help';
import LeakAddIcon from '@mui/icons-material/LeakAdd';
import VpnLockIcon from '@mui/icons-material/VpnLock';
import Box from '@mui/material/Box';
import List from '@mui/material/List';
import ListItem from '@mui/material/ListItem';
import ListItemText from '@mui/material/ListItemText';
import Stack from '@mui/material/Stack';
import Tab from '@mui/material/Tab';
import Tabs from '@mui/material/Tabs';
import Grid from '@mui/material/Unstable_Grid2';
import { ContainerWithHeader } from '../component/ContainerWithHeader';
import { FindPlayerByIP } from '../component/FindPlayerByIP.tsx';
import { FindPlayerIPs } from '../component/FindPlayerIPs.tsx';
import { NetworkBlockChecker } from '../component/NetworkBlockChecker';
import { NetworkBlockSources } from '../component/NetworkBlockSources';
import { NetworkInfo } from '../component/NetworkInfo.tsx';
import { TabPanel } from '../component/TabPanel';

export const AdminNetworkPage = () => {
    const [value, setValue] = useState(0);

    const handleChange = (_: SyntheticEvent, newValue: number) => {
        setValue(newValue);
    };

    return (
        <Grid container padding={0} spacing={2}>
            <Grid xs={9}>
                <ContainerWithHeader
                    title={'Network Tools'}
                    iconLeft={<LeakAddIcon />}
                >
                    <Box sx={{ borderBottom: 1, borderColor: 'divider' }}>
                        <Tabs
                            value={value}
                            onChange={handleChange}
                            aria-label="basic tabs example"
                        >
                            <Tab label="Player IPs" />
                            <Tab label="Find Players By IP" />
                            <Tab label="IP Info" />
                            <Tab label={'External CIDR Bans'} />
                        </Tabs>
                    </Box>
                    <TabPanel index={value} value={0}>
                        <FindPlayerIPs />
                    </TabPanel>
                    <TabPanel value={value} index={1}>
                        <FindPlayerByIP />
                    </TabPanel>
                    <TabPanel value={value} index={2}>
                        <NetworkInfo />
                    </TabPanel>
                    <TabPanel value={value} index={3}>
                        <NetworkBlockSources />
                    </TabPanel>
                </ContainerWithHeader>
            </Grid>
            <Grid xs={3}>
                <Stack spacing={2}>
                    <ContainerWithHeader
                        title={'Tool Overview'}
                        iconLeft={<HelpIcon />}
                    >
                        <List>
                            <ListItem>
                                <ListItemText
                                    primary={'Lookup Player IP'}
                                    secondary={`Query IPs a player has used`}
                                />
                            </ListItem>
                            <ListItem>
                                <ListItemText
                                    primary={'Find Players By IP'}
                                    secondary={`Query players using a particular ip or cidr range.`}
                                />
                            </ListItem>
                            <ListItem>
                                <ListItemText
                                    primary={'IP Info'}
                                    secondary={`Look up metadata for an ip/network`}
                                />
                            </ListItem>
                            <ListItem>
                                <ListItemText
                                    primary={'External CIDR Bans'}
                                    secondary={`Used for banning large range of address blocks using 3rd party URL sources. Response should be in the 
                                format of 1 cidr address per line. Invalid lines are discarded. Use the whitelist to override blocked addresses you want to allow.`}
                                />
                            </ListItem>
                        </List>
                    </ContainerWithHeader>
                    <ContainerWithHeader
                        title={'Blocked IP Checker'}
                        iconLeft={<VpnLockIcon />}
                    >
                        <NetworkBlockChecker />
                    </ContainerWithHeader>
                </Stack>
            </Grid>
        </Grid>
    );
};

export default AdminNetworkPage;
