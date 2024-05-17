import FilterListIcon from '@mui/icons-material/FilterList';
import HowToVoteIcon from '@mui/icons-material/HowToVote';
import Typography from '@mui/material/Typography';
import Grid from '@mui/material/Unstable_Grid2';
import { useForm } from '@tanstack/react-form';
import { useQuery } from '@tanstack/react-query';
import { createFileRoute, useNavigate } from '@tanstack/react-router';
import { createColumnHelper, getCoreRowModel, useReactTable } from '@tanstack/react-table';
import { zodValidator } from '@tanstack/zod-form-adapter';
import { z } from 'zod';
import { apiVotesQuery, VoteResult } from '../api/votes.ts';
import { ContainerWithHeader } from '../component/ContainerWithHeader.tsx';
import { ContainerWithHeaderAndButtons } from '../component/ContainerWithHeaderAndButtons.tsx';
import { DataTable } from '../component/DataTable.tsx';
import { Paginator } from '../component/Paginator.tsx';
import { PersonCell } from '../component/PersonCell.tsx';
import { TableCellBool } from '../component/TableCellBool.tsx';
import { TableHeadingCell } from '../component/TableHeadingCell.tsx';
import { Title } from '../component/Title';
import { Buttons } from '../component/field/Buttons.tsx';
import { TextFieldSimple } from '../component/field/TextFieldSimple.tsx';
import { commonTableSearchSchema, LazyResult, RowsPerPage } from '../util/table.ts';
import { renderDateTime } from '../util/text.tsx';
import { makeSteamidValidatorsOptional } from '../util/validator/makeSteamidValidatorsOptional.ts';

const votesSearchSchema = z.object({
    ...commonTableSearchSchema,
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
    const { success, page, sortColumn, rows, sortOrder, source_id, target_id } = Route.useSearch();
    const { data: votes, isLoading } = useQuery({
        queryKey: ['votes', { success, page, sortColumn, rows, sortOrder, source_id, target_id }],
        queryFn: async () => {
            return apiVotesQuery({
                limit: rows ?? defaultRows,
                offset: (page ?? 0) * (rows ?? defaultRows),
                order_by: sortColumn,
                desc: (sortOrder ?? 'desc') == 'desc',
                source_id: source_id ?? '',
                target_id: target_id ?? '',
                success: success ?? -1
            });
        }
    });

    const { Field, Subscribe, handleSubmit, reset } = useForm({
        onSubmit: async ({ value }) => {
            await navigate({ to: '/admin/votes', search: (prev) => ({ ...prev, ...value }) });
        },
        validatorAdapter: zodValidator,
        validators: {
            onChange: votesSearchSchema
        },
        defaultValues: {
            source_id: source_id ?? '',
            target_id: target_id ?? ''
        }
    });

    const clear = async () => {
        await navigate({
            to: '/admin/votes',
            search: (prev) => ({ ...prev, source_id: undefined, target_id: undefined, success: undefined })
        });
    };

    return (
        <Grid container spacing={2}>
            <Title>Votes</Title>
            <Grid xs={12}>
                <ContainerWithHeader title={'Filters'} iconLeft={<FilterListIcon />} marginTop={2}>
                    <form
                        onSubmit={async (e) => {
                            e.preventDefault();
                            e.stopPropagation();
                            await handleSubmit();
                        }}
                    >
                        <Grid container spacing={2}>
                            <Grid xs={6} md={6}>
                                <Field
                                    name={'source_id'}
                                    validators={makeSteamidValidatorsOptional()}
                                    children={(props) => {
                                        return (
                                            <TextFieldSimple {...props} label={'Initiator Steam ID'} fullwidth={true} />
                                        );
                                    }}
                                />
                            </Grid>

                            <Grid xs={6} md={6}>
                                <Field
                                    name={'target_id'}
                                    validators={makeSteamidValidatorsOptional()}
                                    children={(props) => {
                                        return (
                                            <TextFieldSimple {...props} label={'Target Steam ID'} fullwidth={true} />
                                        );
                                    }}
                                />
                            </Grid>

                            <Grid xs={12} mdOffset="auto">
                                <Subscribe
                                    selector={(state) => [state.canSubmit, state.isSubmitting]}
                                    children={([canSubmit, isSubmitting]) => (
                                        <Buttons
                                            reset={reset}
                                            canSubmit={canSubmit}
                                            isSubmitting={isSubmitting}
                                            onClear={clear}
                                        />
                                    )}
                                />
                            </Grid>
                        </Grid>
                    </form>
                </ContainerWithHeader>
            </Grid>
            <Grid xs={12}>
                <ContainerWithHeaderAndButtons title={'Vote History'} iconLeft={<HowToVoteIcon />}>
                    <VotesTable votes={votes ?? { data: [], count: 0 }} isLoading={isLoading} />
                    <Paginator data={votes} page={page ?? 0} rows={rows ?? defaultRows} path={'/admin/votes'} />
                </ContainerWithHeaderAndButtons>
            </Grid>
        </Grid>
    );
}

const columnHelper = createColumnHelper<VoteResult>();

const VotesTable = ({ votes, isLoading }: { votes: LazyResult<VoteResult>; isLoading: boolean }) => {
    const columns = [
        columnHelper.accessor('source_id', {
            header: () => <TableHeadingCell name={'Initiator'} />,
            cell: (info) => (
                <PersonCell
                    steam_id={votes.data[info.row.index].source_id}
                    personaname={votes.data[info.row.index].source_name}
                    avatar_hash={votes.data[info.row.index].source_avatar_hash}
                />
            )
        }),
        columnHelper.accessor('target_id', {
            header: () => <TableHeadingCell name={'Subject'} />,
            cell: (info) => {
                return (
                    <PersonCell
                        steam_id={votes.data[info.row.index].target_id}
                        personaname={votes.data[info.row.index].target_name}
                        avatar_hash={votes.data[info.row.index].target_avatar_hash}
                    />
                );
            }
        }),
        columnHelper.accessor('success', {
            header: () => <TableHeadingCell name={'Success'} />,
            cell: (info) => <TableCellBool enabled={info.getValue()} />
        }),
        columnHelper.accessor('server_name', {
            header: () => <TableHeadingCell name={'Server'} />,
            cell: (info) => <Typography>{info.getValue()}</Typography>
        }),
        columnHelper.accessor('created_on', {
            header: () => <TableHeadingCell name={'Created'} />,
            cell: (info) => <Typography>{renderDateTime(info.getValue())}</Typography>
        })
    ];

    const table = useReactTable({
        data: votes.data,
        columns: columns,
        getCoreRowModel: getCoreRowModel(),
        manualPagination: true,
        autoResetPageIndex: true
    });

    return <DataTable table={table} isLoading={isLoading} />;
};
