import React, { useCallback, useState } from 'react';
import NiceModal from '@ebay/nice-modal-react';
import AddIcon from '@mui/icons-material/Add';
import GavelIcon from '@mui/icons-material/Gavel';
import Box from '@mui/material/Box';
import Button from '@mui/material/Button';
import Grid from '@mui/material/Unstable_Grid2';
import { CIDRBanRecord } from '../api';
import { ContainerWithHeaderAndButtons } from '../component/ContainerWithHeaderAndButtons';
import { ModalBanCIDR } from '../component/modal';
import { BanCIDRTable } from '../component/table/BanCIDRTable';
import { useUserFlashCtx } from '../contexts/UserFlashCtx';
import { logErr } from '../util/errors';

export const AdminBanCIDRPage = () => {
    const [newCIDRBans, setNewCIDRBans] = useState<CIDRBanRecord[]>([]);
    const { sendFlash } = useUserFlashCtx();

    const onNewBanCIDR = useCallback(async () => {
        try {
            const ban = await NiceModal.show<CIDRBanRecord>(ModalBanCIDR, {});
            setNewCIDRBans((prevState) => {
                return [ban, ...prevState];
            });
            sendFlash(
                'success',
                `Created CIDR ban successfully #${ban.net_id}`
            );
        } catch (e) {
            logErr(e);
        }
    }, [sendFlash]);

    return (
        <Grid container>
            <Grid xs={12} marginBottom={2}>
                <Box></Box>
            </Grid>
            <Grid xs={12}>
                <ContainerWithHeaderAndButtons
                    title={'CIDR Ban History'}
                    marginTop={0}
                    iconLeft={<GavelIcon />}
                    buttons={[
                        <Button
                            key={'btn-cidr'}
                            variant={'contained'}
                            color={'success'}
                            startIcon={<AddIcon />}
                            sx={{ marginRight: 2 }}
                            onClick={onNewBanCIDR}
                        >
                            Create
                        </Button>
                    ]}
                >
                    <BanCIDRTable newBans={newCIDRBans} />
                </ContainerWithHeaderAndButtons>
            </Grid>
        </Grid>
    );
};

export default AdminBanCIDRPage;
