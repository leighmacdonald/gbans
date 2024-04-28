import AddIcon from '@mui/icons-material/Add';
import GavelIcon from '@mui/icons-material/Gavel';
import Box from '@mui/material/Box';
import Button from '@mui/material/Button';
import Grid from '@mui/material/Unstable_Grid2';
import { createFileRoute } from '@tanstack/react-router';
import { z } from 'zod';
import { ContainerWithHeaderAndButtons } from '../component/ContainerWithHeaderAndButtons.tsx';
import { commonTableSearchSchema } from '../util/table.ts';

const banSteamSearchSchema = z.object({
    ...commonTableSearchSchema,
    sortColumn: z.enum(['ban_id', 'source_id', 'target_id', 'deleted', 'reason', 'appeal_state', 'valid_until']).catch('ban_id'),
    source_id: z.string().catch(''),
    target_id: z.string().catch(''),
    //ip: z.string().ip().catch(''),
    deleted: z.boolean().catch(false)
});

export const Route = createFileRoute('/_mod/admin/ban/cidr')({
    component: AdminBanCIDR,
    validateSearch: (search) => banSteamSearchSchema.parse(search)
});

function AdminBanCIDR() {
    // const [newCIDRBans, setNewCIDRBans] = useState<CIDRBanRecord[]>([]);
    // const { sendFlash } = useUserFlashCtx();

    // const onNewBanCIDR = useCallback(async () => {
    //     try {
    //         const ban = await NiceModal.show<CIDRBanRecord>(ModalBanCIDR, {});
    //         setNewCIDRBans((prevState) => {
    //             return [ban, ...prevState];
    //         });
    //         sendFlash('success', `Created CIDR ban successfully #${ban.net_id}`);
    //     } catch (e) {
    //         logErr(e);
    //     }
    // }, [sendFlash]);

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
                            // onClick={onNewBanCIDR}
                        >
                            Create
                        </Button>
                    ]}
                ></ContainerWithHeaderAndButtons>
            </Grid>
        </Grid>
    );
}
