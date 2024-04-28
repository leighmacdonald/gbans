import AddIcon from '@mui/icons-material/Add';
import GavelIcon from '@mui/icons-material/Gavel';
import Button from '@mui/material/Button';
import Grid from '@mui/material/Unstable_Grid2';
import { createFileRoute } from '@tanstack/react-router';
import { z } from 'zod';
import { ContainerWithHeaderAndButtons } from '../component/ContainerWithHeaderAndButtons.tsx';
import { commonTableSearchSchema } from '../util/table.ts';

const banGroupSearchSchema = z.object({
    ...commonTableSearchSchema,
    sortColumn: z.enum(['ban_id', 'source_id', 'target_id', 'deleted', 'reason', 'appeal_state', 'valid_until']).catch('ban_id'),
    source_id: z.string().catch(''),
    target_id: z.string().catch(''),
    group_id: z.number().catch(0),
    deleted: z.boolean().catch(false)
});

export const Route = createFileRoute('/_mod/admin/ban/group')({
    component: AdminBanGroup,
    validateSearch: (search) => banGroupSearchSchema.parse(search)
});

function AdminBanGroup() {
    // const [newGroupBans, setNewGroupBans] = useState<GroupBanRecord[]>([]);
    // const { sendFlash } = useUserFlashCtx();

    // const onNewBanGroup = useCallback(async () => {
    //     try {
    //         const ban = await NiceModal.show<GroupBanRecord>(ModalBanGroup, {});
    //         setNewGroupBans((prevState) => {
    //             return [ban, ...prevState];
    //         });
    //         sendFlash('success', `Created steam group ban successfully #${ban.ban_group_id}`);
    //     } catch (e) {
    //         logErr(e);
    //     }
    // }, [sendFlash]);

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
                            // onClick={onNewBanGroup}
                        >
                            Create
                        </Button>
                    ]}
                >
                    {/*<BanGroupTable newBans={newGroupBans} />*/}
                </ContainerWithHeaderAndButtons>
            </Grid>
        </Grid>
    );
}
