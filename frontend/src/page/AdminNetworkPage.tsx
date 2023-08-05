import React from 'react';
import Grid from '@mui/material/Unstable_Grid2';
import { a11yProps } from '../component/ProfilePanel';
import { TabPanel } from '../component/TabPanel';
import Box from '@mui/material/Box';
import Tabs from '@mui/material/Tabs';
import Tab from '@mui/material/Tab';
import { ContainerWithHeader } from '../component/ContainerWithHeader';
import ListItem from '@mui/material/ListItem';
import List from '@mui/material/List';
import ListItemText from '@mui/material/ListItemText';

export const AdminNetworkPage = () => {
    const [value, setValue] = React.useState(0);

    const handleChange = (_: React.SyntheticEvent, newValue: number) => {
        setValue(newValue);
    };

    return (
        <Grid container padding={0} marginTop={2} spacing={1}>
            <Grid xs={9}>
                <ContainerWithHeader title={'Network Query Tools'}>
                    <Box sx={{ borderBottom: 1, borderColor: 'divider' }}>
                        <Tabs
                            value={value}
                            onChange={handleChange}
                            aria-label="basic tabs example"
                        >
                            <Tab label="Find Players" {...a11yProps(0)} />
                            <Tab label="IP Info" {...a11yProps(1)} />
                        </Tabs>
                    </Box>
                    <TabPanel value={value} index={0}>
                        Find Players
                    </TabPanel>
                    <TabPanel value={value} index={1}>
                        IPInfo
                    </TabPanel>
                </ContainerWithHeader>
            </Grid>
            <Grid xs={3}>
                <ContainerWithHeader title={'Tool Overview'}>
                    <List>
                        <ListItem>
                            <ListItemText
                                primary={'Find Players'}
                                secondary={`Query players using a particular ip or cidr range.`}
                            ></ListItemText>
                        </ListItem>
                        <ListItem>
                            <ListItemText
                                primary={'IP Info'}
                                secondary={`Look up metadata for an ip/network`}
                            ></ListItemText>
                        </ListItem>
                    </List>
                </ContainerWithHeader>
            </Grid>
        </Grid>
    );
};
