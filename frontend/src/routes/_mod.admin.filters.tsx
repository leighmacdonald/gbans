import { useCallback, useMemo, useState } from 'react';
import NiceModal from '@ebay/nice-modal-react';
import AddBoxIcon from '@mui/icons-material/AddBox';
import CancelIcon from '@mui/icons-material/Cancel';
import EditIcon from '@mui/icons-material/Edit';
import FilterAltIcon from '@mui/icons-material/FilterAlt';
import InfoIcon from '@mui/icons-material/Info';
import WarningIcon from '@mui/icons-material/Warning';
import Button from '@mui/material/Button';
import ButtonGroup from '@mui/material/ButtonGroup';
import TableCell from '@mui/material/TableCell';
import Tooltip from '@mui/material/Tooltip';
import Typography from '@mui/material/Typography';
import Grid from '@mui/material/Unstable_Grid2';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { createFileRoute } from '@tanstack/react-router';
import { createColumnHelper } from '@tanstack/react-table';
import { z } from 'zod';
import {
    apiDeleteFilter,
    apiGetFilters,
    apiGetWarningState,
    Filter,
    filterActionString,
    UserWarning
} from '../api/filters.ts';
import { ContainerWithHeader } from '../component/ContainerWithHeader.tsx';
import { ContainerWithHeaderAndButtons } from '../component/ContainerWithHeaderAndButtons.tsx';
import { FullTable } from '../component/FullTable.tsx';
import { IndeterminateCheckbox } from '../component/IndeterminateCheckbox.tsx';
import { PersonCell } from '../component/PersonCell.tsx';
import { TableCellBool } from '../component/TableCellBool.tsx';
import { TableCellSmall } from '../component/TableCellSmall.tsx';
import { TableCellString } from '../component/TableCellString.tsx';
import { TableHeadingCell } from '../component/TableHeadingCell.tsx';
import { Title } from '../component/Title';
import { ModalConfirm, ModalFilterEditor } from '../component/modal';
import { useUserFlashCtx } from '../hooks/useUserFlashCtx.ts';
import { findSelectedRow } from '../util/findSelectedRow.ts';
import { findSelectedRows } from '../util/findSelectedRows.ts';
import { renderDateTime } from '../util/text.tsx';

const filterSearchSchema = z.object({
    sortColumn: z.string().optional(),
    desc: z.boolean().optional()
});

export const Route = createFileRoute('/_mod/admin/filters')({
    component: AdminFilters,
    validateSearch: (search) => filterSearchSchema.parse(search)
});

function AdminFilters() {
    const { sendFlash } = useUserFlashCtx();
    //const { sortColumn, desc } = Route.useSearch();
    const queryClient = useQueryClient();
    const [rowSelection, setRowSelection] = useState({});

    const { data: filters, isLoading } = useQuery({
        queryKey: ['filters'],
        queryFn: async () => {
            return await apiGetFilters();
        }
    });

    const { data: warnings, isLoading: isLoadingWarnings } = useQuery({
        queryKey: ['filterWarnings'],
        queryFn: async () => {
            return await apiGetWarningState();
        }
    });

    const onCreate = useCallback(async () => {
        try {
            const resp = await NiceModal.show<Filter>(ModalFilterEditor, {});
            queryClient.setQueryData(['filters'], [...(filters ?? []), resp]);
        } catch (e) {
            sendFlash('error', `${e}`);
        }
    }, [filters, queryClient, sendFlash]);

    const onEdit = useCallback(async () => {
        try {
            const filter = findSelectedRow(rowSelection, filters ?? []);
            const resp = await NiceModal.show<Filter>(ModalFilterEditor, { filter });

            queryClient.setQueryData(
                ['filters'],
                (filters ?? []).map((f) => {
                    return f.filter_id == resp.filter_id ? resp : f;
                })
            );
        } catch (e) {
            sendFlash('error', `${e}`);
        }
    }, [filters, queryClient, rowSelection, sendFlash]);

    const deleteMutation = useMutation({
        mutationKey: ['filters'],
        mutationFn: async (filter_id: number) => {
            await apiDeleteFilter(filter_id);
        },
        onSuccess: (_, filterId) => {
            sendFlash('error', `Deleted filter: ${filterId}`);
        }
    });

    const onDelete = useCallback(async () => {
        const selectedFiltersIds = findSelectedRows(rowSelection, filters ?? [])?.map((f) => f.filter_id);
        if (!selectedFiltersIds) {
            return;
        }

        try {
            const confirmed = await NiceModal.show(ModalConfirm, {
                title: `Are you sure you want to delete ${selectedFiltersIds.length} filter(s)?`
            });

            if (!confirmed) {
                return;
            }

            selectedFiltersIds.map((f) => {
                deleteMutation.mutate(f as number);
            });
            queryClient.setQueryData(
                ['filters'],
                (filters ?? []).filter((filter) => !selectedFiltersIds.includes(filter.filter_id))
            );
            setRowSelection({});
        } catch (e) {
            sendFlash('error', `${e}`);
            return;
        }
    }, [deleteMutation, filters, queryClient, rowSelection, sendFlash]);

    const columns = useMemo(() => {
        return makeFiltersColumns();
    }, []);

    return (
        <Grid container spacing={2}>
            <Title>Filtered Words</Title>
            <Grid xs={12}>
                <ContainerWithHeaderAndButtons
                    title={`Word Filters ${Object.values(rowSelection).length ? `Selected: ${Object.values(rowSelection).length}` : ''}`}
                    iconLeft={<FilterAltIcon />}
                    buttons={[
                        <ButtonGroup
                            variant="contained"
                            aria-label="outlined primary button group"
                            key={`btn-headers-filters`}
                        >
                            <Button
                                disabled={Object.values(rowSelection).length == 0}
                                color={'error'}
                                onClick={onDelete}
                                startIcon={<CancelIcon />}
                            >
                                Delete
                            </Button>
                            <Button
                                disabled={Object.values(rowSelection).length != 1}
                                color={'warning'}
                                onClick={onEdit}
                                startIcon={<EditIcon />}
                            >
                                Edit
                            </Button>
                            <Button startIcon={<AddBoxIcon />} color={'success'} onClick={onCreate}>
                                New
                            </Button>
                        </ButtonGroup>
                    ]}
                >
                    <FullTable data={filters ?? []} isLoading={isLoading} columns={columns} />
                </ContainerWithHeaderAndButtons>
            </Grid>
            <Grid xs={12}>
                <ContainerWithHeader
                    title={`Current Warning State (Max Weight: ${warnings?.max_weight ?? '...'})`}
                    iconLeft={<WarningIcon />}
                >
                    <WarningStateTable warnings={warnings?.current ?? []} isLoading={isLoadingWarnings} />
                </ContainerWithHeader>
            </Grid>
            <Grid xs={12}>
                <ContainerWithHeader title={'How it works'} iconLeft={<InfoIcon />}>
                    <Typography variant={'body1'}>
                        The way the warning tracking works is that each time a user triggers a match, it gets a entry in
                        the table based on the weight of the match. The individual match weight is determined by the
                        word filter defined above. Once the sum of their triggers exceeds the max weight the user will
                        have action taken against them automatically. Matched entries are ephemeral and are removed over
                        time based on the configured timeout value.
                    </Typography>
                </ContainerWithHeader>
            </Grid>
        </Grid>
    );
}

const makeFiltersColumns = () => {
    const columnHelper = createColumnHelper<Filter>();

    return [
        columnHelper.display({
            id: 'select',
            header: ({ table }) => (
                <IndeterminateCheckbox
                    {...{
                        checked: table.getIsAllRowsSelected(),
                        indeterminate: table.getIsSomeRowsSelected(),
                        onChange: table.getToggleAllRowsSelectedHandler()
                    }}
                />
            ),
            cell: ({ row }) => (
                <div className="px-1">
                    <IndeterminateCheckbox
                        {...{
                            checked: row.getIsSelected(),
                            disabled: !row.getCanSelect(),
                            indeterminate: row.getIsSomeSelected(),
                            onChange: row.getToggleSelectedHandler()
                        }}
                    />
                </div>
            )
        }),
        columnHelper.accessor('pattern', {
            cell: (info) => info.getValue(),
            enableSorting: true
        }),
        columnHelper.accessor('is_regex', {
            cell: (info) => <TableCellBool enabled={info.getValue() as boolean} />,
            header: () => <TableHeadingCell name={'Rx'} />,
            enableSorting: true
        }),
        columnHelper.accessor('action', {
            cell: (info) => {
                return (
                    <TableCellString>
                        {typeof info.row.original === 'undefined' ? '' : filterActionString(info.row.original.action)}
                    </TableCellString>
                );
            },
            header: () => <TableHeadingCell name={'Action'} />,
            enableSorting: true
        }),
        columnHelper.accessor('duration', {
            cell: (info) => <TableCellString>{info.getValue()}</TableCellString>,
            header: () => <TableHeadingCell name={'Duration'} />,
            enableSorting: true
        }),
        columnHelper.accessor('weight', {
            cell: (info) => <TableCellString>{info.getValue()}</TableCellString>,
            header: () => <TableHeadingCell name={'Weight'} />,
            enableSorting: true
        }),
        columnHelper.accessor('trigger_count', {
            cell: (info) => <TableCellString>{info.getValue()}</TableCellString>,
            header: () => <TableHeadingCell name={'Enabled'} />,
            enableSorting: true
        })
    ];
};

export const WarningStateTable = ({ warnings, isLoading }: { warnings: UserWarning[]; isLoading: boolean }) => {
    const renderFilter = (f: Filter) => {
        const pat = f.is_regex ? (f.pattern as string) : (f.pattern as string);

        return (
            <>
                <Typography variant={'h6'}>Matched {f.is_regex ? 'Regex' : 'Text'}</Typography>
                <Typography variant={'body1'}>{pat}</Typography>
                <Typography variant={'body1'}>Weight: {f.weight}</Typography>
                <Typography variant={'body1'}>Action: {filterActionString(f.action)}</Typography>
            </>
        );
    };
    const columnHelper = createColumnHelper<UserWarning>();

    const columns = [
        columnHelper.accessor('steam_id', {
            header: () => <TableHeadingCell name={'Profile'} />,
            cell: (info) => (
                <TableCellSmall>
                    <PersonCell
                        steam_id={info.getValue()}
                        personaname={warnings[info.row.index].personaname}
                        avatar_hash={warnings[info.row.index].avatar}
                    />
                </TableCellSmall>
            )
        }),
        columnHelper.accessor('created_on', {
            header: () => <TableHeadingCell name={'Triggered At'} />,
            cell: (info) => <TableCellString>{renderDateTime(info.getValue())}</TableCellString>
        }),
        columnHelper.accessor('server_name', {
            header: () => <TableHeadingCell name={'Server'} />,
            cell: (info) => (
                <TableCellSmall>
                    <Typography>{info.getValue()}</Typography>
                </TableCellSmall>
            )
        }),
        columnHelper.accessor('matched', {
            header: () => <TableHeadingCell name={'Duration'} />,
            cell: (info) => (
                <TableCell>
                    <Tooltip title={renderFilter(warnings[info.row.index].matched_filter)}>
                        <Typography>{info.getValue()}</Typography>
                    </Tooltip>
                </TableCell>
            )
        }),
        columnHelper.accessor('current_total', {
            header: () => <TableHeadingCell name={'Weight'} />,
            cell: (info) => <TableCellString>{info.getValue()}</TableCellString>
        }),
        columnHelper.accessor('message', {
            header: () => <TableHeadingCell name={'Trigger Message'} />,
            cell: (info) => <TableCellString>{info.getValue()}</TableCellString>
        })
    ];

    return <FullTable data={warnings ?? []} isLoading={isLoading} columns={columns} />;
};
