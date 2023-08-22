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
import TextField from '@mui/material/TextField';
import IPCIDR from 'ip-cidr';

interface NetworkInputProps {
    onValidChange: (cidr: string) => void;
}

export const NetworkInput = ({ onValidChange }: NetworkInputProps) => {
    const defaultHelperText = 'Enter a IP address or CIDR range';
    const [error, setError] = React.useState('');
    const [value, setValue] = React.useState('');
    const [helper, setHelper] = React.useState(defaultHelperText);

    const onChange = React.useCallback(
        (evt: React.ChangeEvent<HTMLInputElement>) => {
            const address = evt.target.value;
            if (address == '') {
                setError('');
                setValue(address);
                setHelper(defaultHelperText);
                return;
            }
            if (!address.match(`^([0-9./]+?)$`)) {
                return;
            }

            setValue(address);

            if (address.length > 0 && !IPCIDR.isValidAddress(address)) {
                setError('Invalid address');
                return;
            }

            setError('');

            try {
                const cidr = new IPCIDR(address);
                setHelper(`Total hosts in range: ${cidr.size}`);
                onValidChange(address);
            } catch (e) {
                if (IPCIDR.isValidAddress(address)) {
                    setHelper(`Total hosts in range: 1`);
                    onValidChange(address);
                }
                return;
            }
        },
        [onValidChange]
    );

    return (
        <TextField
            fullWidth
            error={Boolean(error.length)}
            id="outlined-error-helper-text"
            label="IP/CIDR"
            value={value}
            onChange={onChange}
            helperText={helper}
        />
    );
};

const FindPlayerIP = () => {
    return (
        <Grid container>
            <Grid xs={12}>
                <NetworkInput
                    onValidChange={(cidr) => {
                        console.log(cidr);
                    }}
                />
            </Grid>
        </Grid>
    );
};

export const AdminNetworkPage = () => {
    const [value, setValue] = React.useState(0);

    const handleChange = (_: React.SyntheticEvent, newValue: number) => {
        setValue(newValue);
    };

    return (
        <Grid container padding={0} spacing={2}>
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
                        <FindPlayerIP />
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
