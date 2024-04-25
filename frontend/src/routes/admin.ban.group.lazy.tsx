import { useCallback, useState } from 'react';
import NiceModal from '@ebay/nice-modal-react';
import AddIcon from '@mui/icons-material/Add';
import GavelIcon from '@mui/icons-material/Gavel';
import Button from '@mui/material/Button';
import Grid from '@mui/material/Unstable_Grid2';
import { createLazyFileRoute } from '@tanstack/react-router';
import { GroupBanRecord } from '../api';
import { ContainerWithHeaderAndButtons } from '../component/ContainerWithHeaderAndButtons';
import { ModalBanGroup } from '../component/modal';
import { BanGroupTable } from '../component/table/BanGroupTable';
import { useUserFlashCtx } from '../hooks/useUserFlashCtx.ts';
import { logErr } from '../util/errors';

export const Route = createLazyFileRoute('/admin/ban/group')({
    component: AdminBanGroup
});

function AdminBanGroup() {
    const [newGroupBans, setNewGroupBans] = useState<GroupBanRecord[]>([]);
    const { sendFlash } = useUserFlashCtx();

    const onNewBanGroup = useCallback(async () => {
        try {
            const ban = await NiceModal.show<GroupBanRecord>(ModalBanGroup, {});
            setNewGroupBans((prevState) => {
                return [ban, ...prevState];
            });
            sendFlash(
                'success',
                `Created steam group ban successfully #${ban.ban_group_id}`
            );
        } catch (e) {
            logErr(e);
        }
    }, [sendFlash]);

    return (
        <Grid container>
            <Grid xs={12}>
                <ContainerWithHeaderAndButtons
                    title={'Steam Group Ban History'}
                    marginTop={0}
                    iconLeft={<GavelIcon />}
                    buttons={[
                        <Button
                            key={`ban-group`}
                            variant={'contained'}
                            color={'success'}
                            startIcon={<AddIcon />}
                            sx={{ marginRight: 2 }}
                            onClick={onNewBanGroup}
                        >
                            Create
                        </Button>
                    ]}
                >
                    <BanGroupTable newBans={newGroupBans} />
                </ContainerWithHeaderAndButtons>
            </Grid>
        </Grid>
    );
}
