import { useCallback, useState } from 'react';
import NiceModal from '@ebay/nice-modal-react';
import AddIcon from '@mui/icons-material/Add';
import GavelIcon from '@mui/icons-material/Gavel';
import Button from '@mui/material/Button';
import Grid from '@mui/material/Unstable_Grid2';
import { ASNBanRecord } from '../api';
import { ContainerWithHeaderAndButtons } from '../component/ContainerWithHeaderAndButtons';
import { ModalBanASN } from '../component/modal';
import { BanASNTable } from '../component/table/BanASNTable';
import { useUserFlashCtx } from '../contexts/UserFlashCtx';
import { logErr } from '../util/errors';

export const AdminBanASNPage = () => {
    const [newASNBans, setNewASNBans] = useState<ASNBanRecord[]>([]);
    const { sendFlash } = useUserFlashCtx();

    const onNewBanASN = useCallback(async () => {
        try {
            const ban = await NiceModal.show<ASNBanRecord>(ModalBanASN, {});
            setNewASNBans((prevState) => {
                return [ban, ...prevState];
            });
            sendFlash(
                'success',
                `Created ASN ban successfully #${ban.ban_asn_id}`
            );
        } catch (e) {
            logErr(e);
        }
    }, [sendFlash]);

    return (
        <Grid container>
            <Grid xs={12}>
                <ContainerWithHeaderAndButtons
                    title={'ASN Ban History'}
                    marginTop={0}
                    iconLeft={<GavelIcon />}
                    buttons={[
                        <Button
                            key={'btn-asn'}
                            variant={'contained'}
                            color={'success'}
                            startIcon={<AddIcon />}
                            sx={{ marginRight: 2 }}
                            onClick={onNewBanASN}
                        >
                            Create
                        </Button>
                    ]}
                >
                    <BanASNTable newBans={newASNBans} />
                </ContainerWithHeaderAndButtons>
            </Grid>
        </Grid>
    );
};

export default AdminBanASNPage;
