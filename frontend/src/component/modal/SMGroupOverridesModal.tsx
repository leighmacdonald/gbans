import { useMemo } from 'react';
import NiceModal, { muiDialogV5, useModal } from '@ebay/nice-modal-react';
import CloseIcon from '@mui/icons-material/Close';
import GroupsIcon from '@mui/icons-material/Groups';
import { Dialog, DialogActions, DialogContent, DialogTitle } from '@mui/material';
import Button from '@mui/material/Button';
import Grid from '@mui/material/Unstable_Grid2';
import { useQuery } from '@tanstack/react-query';
import { createColumnHelper } from '@tanstack/react-table';
import 'video-react/dist/video-react.css';
import { apiGetSMGroupOverrides, SMGroups, SMOverrides } from '../../api';
import { renderDateTime } from '../../util/text.tsx';
import { FullTable } from '../FullTable.tsx';
import { Heading } from '../Heading';
import { TableCellString } from '../TableCellString.tsx';
import { TableHeadingCell } from '../TableHeadingCell.tsx';

const overrideColumnHelper = createColumnHelper<SMOverrides>();

const makeColumns = () => [
    overrideColumnHelper.accessor('name', {
        header: () => <TableHeadingCell name={'Name'} />,
        cell: (info) => <TableCellString>{info.getValue()}</TableCellString>
    }),
    overrideColumnHelper.accessor('type', {
        header: () => <TableHeadingCell name={'Type'} />,
        cell: (info) => <TableCellString>{info.getValue()}</TableCellString>
    }),
    overrideColumnHelper.accessor('flags', {
        header: () => <TableHeadingCell name={'Flags'} />,
        cell: (info) => <TableCellString>{info.getValue()}</TableCellString>
    }),
    overrideColumnHelper.accessor('created_on', {
        header: () => <TableHeadingCell name={'Created'} />,
        cell: (info) => <TableCellString>{renderDateTime(info.getValue())}</TableCellString>
    }),
    overrideColumnHelper.accessor('updated_on', {
        header: () => <TableHeadingCell name={'Updated'} />,
        cell: (info) => <TableCellString>{renderDateTime(info.getValue())}</TableCellString>
    })
];

export const SMGroupOverridesModal = NiceModal.create(({ group }: { group: SMGroups }) => {
    const modal = useModal();

    const { data: overrides, isLoading } = useQuery({
        queryKey: ['serverGroupOverrides', { group_id: group.group_id }],
        queryFn: async () => {
            return await apiGetSMGroupOverrides(group.group_id);
        }
    });

    const columns = useMemo(() => {
        return makeColumns();
    }, []);

    return (
        <Dialog fullWidth {...muiDialogV5(modal)}>
            <DialogTitle component={Heading} iconLeft={<GroupsIcon />}>
                Group Overrides
            </DialogTitle>

            <DialogContent>
                <FullTable data={overrides ?? []} isLoading={isLoading} columns={columns} />
            </DialogContent>

            <DialogActions>
                <Grid container>
                    <Grid xs={12} mdOffset="auto">
                        <Button
                            key={'close-button'}
                            onClick={modal.hide}
                            variant={'contained'}
                            color={'error'}
                            startIcon={<CloseIcon />}
                        >
                            Close
                        </Button>
                    </Grid>
                </Grid>
            </DialogActions>
        </Dialog>
    );
});
