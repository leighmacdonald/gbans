import React, { useCallback } from 'react';
import NiceModal from '@ebay/nice-modal-react';
import DirectionsRunIcon from '@mui/icons-material/DirectionsRun';
import GavelIcon from '@mui/icons-material/Gavel';
import GroupsIcon from '@mui/icons-material/Groups';
import LanIcon from '@mui/icons-material/Lan';
import RouterIcon from '@mui/icons-material/Router';
import TabContext from '@mui/lab/TabContext';
import TabList from '@mui/lab/TabList';
import TabPanel from '@mui/lab/TabPanel';
import Box from '@mui/material/Box';
import Button from '@mui/material/Button';
import ButtonGroup from '@mui/material/ButtonGroup';
import Tab from '@mui/material/Tab';
import Grid from '@mui/material/Unstable_Grid2';
import { useTheme } from '@mui/material/styles';
import {
    ASNBanRecord,
    CIDRBanRecord,
    GroupBanRecord,
    SteamBanRecord
} from '../api';
import { BanASNTable } from '../component/BanASNTable';
import { BanCIDRTable } from '../component/BanCIDRTable';
import { BanGroupTable } from '../component/BanGroupTable';
import { BanSteamTable } from '../component/BanSteamTable';
import { ContainerWithHeader } from '../component/ContainerWithHeader';
import {
    ModalBanASN,
    ModalBanCIDR,
    ModalBanGroup,
    ModalBanSteam
} from '../component/modal';
import { useUserFlashCtx } from '../contexts/UserFlashCtx';

export const AdminBanPage = () => {
    const theme = useTheme();
    const [value, setValue] = React.useState<string>('0');
    const { sendFlash } = useUserFlashCtx();

    const handleChange = (_: React.SyntheticEvent, newValue: string) => {
        setValue(newValue);
    };

    const onNewBanSteam = useCallback(async () => {
        try {
            const ban = await NiceModal.show<SteamBanRecord>(ModalBanSteam, {});
            sendFlash(
                'success',
                `Created steam ban successfully #${ban.ban_id}`
            );
        } catch (e) {
            sendFlash('error', `Failed to save ban: ${e}`);
        }
    }, [sendFlash]);

    const onNewBanCIDR = useCallback(async () => {
        try {
            const ban = await NiceModal.show<CIDRBanRecord>(ModalBanCIDR, {});
            sendFlash(
                'success',
                `Created CIDR ban successfully #${ban.net_id}`
            );
        } catch (e) {
            sendFlash('error', `Failed to save CIDR ban: ${e}`);
        }
    }, [sendFlash]);

    const onNewBanASN = useCallback(async () => {
        try {
            const ban = await NiceModal.show<ASNBanRecord>(ModalBanASN, {});
            sendFlash(
                'success',
                `Created ASN ban successfully #${ban.ban_asn_id}`
            );
        } catch (e) {
            sendFlash('error', `Failed to save ASN ban: ${e}`);
        }
    }, [sendFlash]);

    const onNewBanGroup = useCallback(async () => {
        try {
            const ban = await NiceModal.show<GroupBanRecord>(ModalBanGroup, {});
            sendFlash(
                'success',
                `Created steam group ban successfully #${ban.ban_group_id}`
            );
        } catch (e) {
            sendFlash('error', `Failed to save group ban: ${e}`);
        }
    }, [sendFlash]);

    return (
        <Grid container>
            <Grid xs={12} marginBottom={2}>
                <Box>
                    <ButtonGroup>
                        <Button
                            variant={'contained'}
                            color={'secondary'}
                            startIcon={<DirectionsRunIcon />}
                            sx={{ marginRight: 2 }}
                            onClick={onNewBanSteam}
                        >
                            Steam
                        </Button>
                        <Button
                            variant={'contained'}
                            color={'secondary'}
                            startIcon={<RouterIcon />}
                            sx={{ marginRight: 2 }}
                            onClick={onNewBanCIDR}
                        >
                            CIDR
                        </Button>
                        <Button
                            variant={'contained'}
                            color={'secondary'}
                            startIcon={<LanIcon />}
                            sx={{ marginRight: 2 }}
                            onClick={onNewBanASN}
                        >
                            ASN
                        </Button>
                        <Button
                            variant={'contained'}
                            color={'secondary'}
                            startIcon={<GroupsIcon />}
                            sx={{ marginRight: 2 }}
                            onClick={onNewBanGroup}
                        >
                            Group
                        </Button>
                    </ButtonGroup>
                </Box>
            </Grid>
            <Grid xs={12}>
                <ContainerWithHeader
                    title={'Ban History'}
                    marginTop={0}
                    iconLeft={<GavelIcon />}
                >
                    <TabContext value={value}>
                        <Box
                            sx={{
                                borderBottom: 1,
                                borderColor: 'divider',
                                backgroundColor: theme.palette.background.paper
                            }}
                        >
                            <TabList onChange={handleChange}>
                                <Tab
                                    label={'Steam Bans'}
                                    color={'text'}
                                    icon={<DirectionsRunIcon />}
                                    iconPosition={'start'}
                                    value={'0'}
                                />
                                <Tab
                                    label={`CIDR Bans`}
                                    icon={<RouterIcon />}
                                    iconPosition={'start'}
                                    value={'1'}
                                />
                                <Tab
                                    label={`ASN Bans`}
                                    icon={<LanIcon />}
                                    iconPosition={'start'}
                                    value={'2'}
                                />
                                <Tab
                                    label={`Group Bans`}
                                    icon={<GroupsIcon />}
                                    iconPosition={'start'}
                                    value={'3'}
                                />
                            </TabList>
                        </Box>

                        <TabPanel value={value} sx={{ padding: 0 }}>
                            <div
                                style={{
                                    padding: 0,
                                    margin: 0,
                                    display: '0' == value ? 'block' : 'none'
                                }}
                            >
                                <BanSteamTable />
                            </div>
                            <div
                                style={{
                                    display: '1' == value ? 'block' : 'none'
                                }}
                            >
                                <BanCIDRTable />
                            </div>

                            <div
                                style={{
                                    display: '2' == value ? 'block' : 'none'
                                }}
                            >
                                <BanASNTable />
                            </div>

                            <div
                                style={{
                                    display: '3' == value ? 'block' : 'none'
                                }}
                            >
                                <BanGroupTable />
                            </div>
                        </TabPanel>
                    </TabContext>
                </ContainerWithHeader>
            </Grid>
        </Grid>
    );
};
