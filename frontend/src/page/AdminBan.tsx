import React, { useCallback } from 'react';
import NiceModal from '@ebay/nice-modal-react';
import DirectionsRunIcon from '@mui/icons-material/DirectionsRun';
import GavelIcon from '@mui/icons-material/Gavel';
import GroupsIcon from '@mui/icons-material/Groups';
import LanIcon from '@mui/icons-material/Lan';
import RouterIcon from '@mui/icons-material/Router';
import Box from '@mui/material/Box';
import Button from '@mui/material/Button';
import ButtonGroup from '@mui/material/ButtonGroup';
import Tab from '@mui/material/Tab';
import Tabs from '@mui/material/Tabs';
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
import { TabPanel } from '../component/TabPanel';
import {
    ModalBanASN,
    ModalBanCIDR,
    ModalBanGroup,
    ModalBanSteam
} from '../component/modal';
import { useUserFlashCtx } from '../contexts/UserFlashCtx';

export const AdminBan = () => {
    const theme = useTheme();
    const [value, setValue] = React.useState<number>(0);
    const { sendFlash } = useUserFlashCtx();

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
                    <Box
                        sx={{
                            borderBottom: 1,
                            borderColor: 'divider',
                            backgroundColor: theme.palette.background.paper
                        }}
                    >
                        <Tabs
                            value={value}
                            onChange={(
                                _: React.SyntheticEvent,
                                newValue: number
                            ) => {
                                setValue(newValue);
                            }}
                            aria-label="ReportCreatePage detail tabs"
                        >
                            <Tab label={'Steam Bans'} color={'text'} />
                            <Tab label={`CIDR Bans`} />
                            <Tab label={`ASN Bans`} />
                            <Tab label={`Group Bans`} />
                        </Tabs>
                    </Box>

                    <TabPanel value={value} index={0}>
                        <BanSteamTable />
                    </TabPanel>

                    <TabPanel value={value} index={1}>
                        <BanCIDRTable />
                    </TabPanel>

                    <TabPanel value={value} index={2}>
                        <BanASNTable />
                    </TabPanel>

                    <TabPanel value={value} index={3}>
                        <BanGroupTable />
                    </TabPanel>
                </ContainerWithHeader>
            </Grid>
        </Grid>
    );
};
