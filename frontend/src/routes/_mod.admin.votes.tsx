import { useMemo, useState } from 'react';
import FilterListIcon from '@mui/icons-material/FilterList';
import HowToVoteIcon from '@mui/icons-material/HowToVote';
import ButtonGroup from '@mui/material/ButtonGroup';
import Grid from '@mui/material/Grid';
import Typography from '@mui/material/Typography';
import { useQuery } from '@tanstack/react-query';
import { createFileRoute, useNavigate } from '@tanstack/react-router';
import { createColumnHelper, PaginationState } from '@tanstack/react-table';
import { z } from 'zod/v4';
import { apiVotesQuery } from '../api/votes.ts';
import { ContainerWithHeader } from '../component/ContainerWithHeader.tsx';
import { ContainerWithHeaderAndButtons } from '../component/ContainerWithHeaderAndButtons.tsx';
import { PersonCell } from '../component/PersonCell.tsx';
import { Title } from '../component/Title';
import { FullTable } from '../component/table/FullTable.tsx';
import { TableCellBool } from '../component/table/TableCellBool.tsx';
import { useAppForm } from '../contexts/formContext.tsx';
import { VoteResult } from '../schema/votes.ts';
import { commonTableSearchSchema, initPagination, RowsPerPage } from '../util/table.ts';
import { renderDateTime } from '../util/time.ts';

const votesSearchSchema = commonTableSearchSchema.extend({
    sortColumn: z.enum(['target_id', 'source_id', 'success', 'created_on']).optional(),
    source_id: z.string().optional(),
    target_id: z.string().optional(),
    success: z.number().optional()
});

export const Route = createFileRoute('/_mod/admin/votes')({
    component: AdminVotes,
    validateSearch: (search) => votesSearchSchema.parse(search)
});

function AdminVotes() {
    const defaultRows = RowsPerPage.TwentyFive;
    const navigate = useNavigate({ from: Route.fullPath });
    const search = Route.useSearch();
    const [pagination, setPagination] = useState<PaginationState>(initPagination(search.pageIndex, search.pageSize));

    const { data: votes, isLoading } = useQuery({
        queryKey: ['votes', { search }],
        queryFn: async () => {
            return apiVotesQuery({
                limit: search.pageSize ?? defaultRows,
                offset: (search.pageIndex ?? 0) * (search.pageSize ?? defaultRows),
                order_by: search.sortColumn ?? 'vote_id',
                desc: (search.sortOrder ?? 'desc') == 'desc',
                source_id: search.source_id ?? '',
                target_id: search.target_id ?? '',
                success: search.success ?? -1
            });
        }
    });

    const form = useAppForm({
        onSubmit: async ({ value }) => {
            await navigate({ to: '/admin/votes', search: (prev) => ({ ...prev, ...value }) });
        },
        validators: {
            onChange: z.object({
                source_id: z.string(),
                target_id: z.string()
            })
        },
        defaultValues: {
            source_id: search.source_id ?? '',
            target_id: search.target_id ?? ''
        }
    });

    const clear = async () => {
        form.reset();
        await navigate({
            to: '/admin/votes',
            search: (prev) => ({ ...prev, source_id: undefined, target_id: undefined, success: undefined })
        });
    };

    const columns = useMemo(() => {
        return makeVoteColumns();
    }, []);

    return (
        <Grid container spacing={2}>
            <Title>Votes</Title>
            <Grid size={{ xs: 12 }}>
                <ContainerWithHeader title={'Filters'} iconLeft={<FilterListIcon />} marginTop={2}>
                    <form
                        onSubmit={async (e) => {
                            e.preventDefault();
                            e.stopPropagation();
                            await form.handleSubmit();
                        }}
                    >
                        <Grid container spacing={2}>
                            <Grid size={{ xs: 6, md: 6 }}>
                                <form.AppField
                                    name={'source_id'}
                                    children={(field) => {
                                        return <field.SteamIDField label={'Initiator Steam ID'} />;
                                    }}
                                />
                            </Grid>

                            <Grid size={{ xs: 6, md: 6 }}>
                                <form.AppField
                                    name={'target_id'}
                                    children={(field) => {
                                        return <field.SteamIDField label={'Target Steam ID'} />;
                                    }}
                                />
                            </Grid>

                            <Grid size={{ xs: 12 }}>
                                <form.AppForm>
                                    <ButtonGroup>
                                        <form.ClearButton onClick={clear} />
                                        <form.ResetButton />
                                        <form.SubmitButton />
                                    </ButtonGroup>
                                </form.AppForm>
                            </Grid>
                        </Grid>
                    </form>
                </ContainerWithHeader>
            </Grid>
            <Grid size={{ xs: 12 }}>
                <ContainerWithHeaderAndButtons title={'Vote History'} iconLeft={<HowToVoteIcon />}>
                    <FullTable
                        data={votes?.data ?? []}
                        isLoading={isLoading}
                        columns={columns}
                        infinitePage={true}
                        pagination={pagination}
                        setPagination={setPagination}
                        toOptions={{ from: Route.fullPath }}
                    />
                </ContainerWithHeaderAndButtons>
            </Grid>
        </Grid>
    );
}

const columnHelper = createColumnHelper<VoteResult>();

const makeVoteColumns = () => {
    return [
        columnHelper.accessor('source_id', {
            header: 'Initiator',
            cell: (info) => (
                <PersonCell
                    showCopy={true}
                    steam_id={info.row.original.source_id}
                    personaname={info.row.original.source_name}
                    avatar_hash={info.row.original.source_avatar_hash}
                />
            )
        }),
        columnHelper.accessor('target_id', {
            header: 'Subject',
            cell: (info) => {
                return (
                    <PersonCell
                        showCopy={true}
                        steam_id={info.row.original.target_id}
                        personaname={info.row.original.target_name}
                        avatar_hash={info.row.original.target_avatar_hash}
                    />
                );
            }
        }),
        columnHelper.accessor('success', {
            header: 'Passed',
            size: 50,
            cell: (info) => <TableCellBool enabled={info.getValue()} />
        }),
        columnHelper.accessor('server_name', {
            header: 'Server',
            size: 75,
            cell: (info) => <Typography>{info.getValue()}</Typography>
        }),
        columnHelper.accessor('created_on', {
            header: 'Created',
            size: 120,
            cell: (info) => <Typography>{renderDateTime(info.getValue())}</Typography>
        })
    ];
};
