import React from 'react';
import Box from '@mui/material/Box';
import List from '@mui/material/List';
import ListItem from '@mui/material/ListItem';
import ListItemText from '@mui/material/ListItemText';
import Tab from '@mui/material/Tab';
import Tabs from '@mui/material/Tabs';
import TextField from '@mui/material/TextField';
import Grid from '@mui/material/Unstable_Grid2';
import IPCIDR from 'ip-cidr';
import { ContainerWithHeader } from '../component/ContainerWithHeader';
import { TabPanel } from '../component/TabPanel';

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
                            <Tab label="Find Players" />
                            <Tab label="IP Info" />
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
