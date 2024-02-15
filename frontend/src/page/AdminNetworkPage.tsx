import { ChangeEvent, SyntheticEvent, useCallback, useState } from 'react';
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
import TextField from '@mui/material/TextField';
import Grid from '@mui/material/Unstable_Grid2';
import IPCIDR from 'ip-cidr';
import { ContainerWithHeader } from '../component/ContainerWithHeader';
import { NetworkBlockChecker } from '../component/NetworkBlockChecker';
import { NetworkBlockSources } from '../component/NetworkBlockSources';
import { TabPanel } from '../component/TabPanel';

interface NetworkInputProps {
    onValidChange: (cidr: string) => void;
}

export const NetworkInput = ({ onValidChange }: NetworkInputProps) => {
    const defaultHelperText = 'Enter a IP address or CIDR range';
    const [error, setError] = useState('');
    const [value, setValue] = useState('');
    const [helper, setHelper] = useState(defaultHelperText);

    const onChange = useCallback(
        (evt: ChangeEvent<HTMLInputElement>) => {
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
                            <Tab label="Find Players" />
                            <Tab label="IP Info" />
                            <Tab label={'External CIDR Bans'} />
                        </Tabs>
                    </Box>
                    <TabPanel value={value} index={0}>
                        <FindPlayerIP />
                    </TabPanel>
                    <TabPanel value={value} index={1}>
                        IPInfo
                    </TabPanel>
                    <TabPanel value={value} index={2}>
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
                                    primary={'Find Players'}
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
