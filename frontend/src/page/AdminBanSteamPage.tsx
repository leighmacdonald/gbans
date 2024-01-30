import React, { useCallback, useState } from 'react';
import NiceModal from '@ebay/nice-modal-react';
import AddIcon from '@mui/icons-material/Add';
import GavelIcon from '@mui/icons-material/Gavel';
import Button from '@mui/material/Button';
import Grid from '@mui/material/Unstable_Grid2';
import { SteamBanRecord } from '../api';
import { ContainerWithHeaderAndButtons } from '../component/ContainerWithHeaderAndButtons';
import { ModalBanSteam } from '../component/modal';
import { BanSteamTable } from '../component/table/BanSteamTable';
import { useUserFlashCtx } from '../contexts/UserFlashCtx';
import { logErr } from '../util/errors';

export const AdminBanSteamPage = () => {
    const [newSteamBans, setNewSteamBans] = useState<SteamBanRecord[]>([]);
    const { sendFlash } = useUserFlashCtx();

    const onNewBanSteam = useCallback(async () => {
        try {
            const ban = await NiceModal.show<SteamBanRecord>(ModalBanSteam, {});
            setNewSteamBans((prevState) => {
                return [ban, ...prevState];
            });
            sendFlash(
                'success',
                `Created steam ban successfully #${ban.ban_id}`
            );
        } catch (e) {
            logErr(e);
        }
    }, [sendFlash]);

    return (
        <Grid container>
            <Grid xs={12}>
                <ContainerWithHeaderAndButtons
                    title={'Steam Ban History'}
                    marginTop={0}
                    iconLeft={<GavelIcon />}
                    buttons={[
                        <Button
                            key={`ban-steam`}
                            variant={'contained'}
                            color={'success'}
                            startIcon={<AddIcon />}
                            sx={{ marginRight: 2 }}
                            onClick={onNewBanSteam}
                        >
                            Create
                        </Button>
                    ]}
                >
                    <BanSteamTable newBans={newSteamBans} />
                </ContainerWithHeaderAndButtons>
            </Grid>
        </Grid>
    );
};

export default AdminBanSteamPage;
