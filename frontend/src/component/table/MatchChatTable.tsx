import React from 'react';
import useUrlState from '@ahooksjs/use-url-state';
import FlagIcon from '@mui/icons-material/Flag';
import Typography from '@mui/material/Typography';
import Grid from '@mui/material/Unstable_Grid2';
import { formatISO9075 } from 'date-fns/fp';
import { defaultAvatarHash, PersonMessage } from '../../api';
import { useChatHistory } from '../../hooks/useChatHistory';
import { ChatContextMenu } from '../ChatContextMenu';
import { LoadingPlaceholder } from '../LoadingPlaceholder';
import { PersonCellFieldNonInteractive } from '../formik/PersonCellField';
import { LazyTable, RowsPerPage } from './LazyTable';

export const MatchChatTable = ({ match_id }: { match_id: string }) => {
    const [state, setState] = useUrlState({
        sortOrder: undefined,
        sortColumn: undefined,
        page: undefined,
        rows: undefined
    });

    const {
        data: messages,
        count,
        loading
    } = useChatHistory({
        match_id: match_id,
        limit: Number(state.rows ?? RowsPerPage.TwentyFive),
        offset: Number((state.page ?? 0) * (state.rows ?? RowsPerPage.Ten)),
        order_by: state.sortColumn ?? 'person_message_id',
        desc: (state.sortOrder ?? 'asc') == 'desc'
    });

    return loading || match_id == '' ? (
        <LoadingPlaceholder />
    ) : (
        <LazyTable<PersonMessage>
            showPager
            rows={messages}
            page={Number(state.page ?? '0')}
            count={count}
            rowsPerPage={Number(state.rows ?? RowsPerPage.TwentyFive)}
            sortOrder={state.sortOrder}
            sortColumn={state.sortColumn}
            onSortColumnChanged={async (column) => {
                setState({ sortColumn: column });
            }}
            onSortOrderChanged={async (direction) => {
                setState({ sortOrder: direction });
            }}
            onRowsPerPageChange={(
                event: React.ChangeEvent<HTMLInputElement | HTMLTextAreaElement>
            ) => {
                setState({
                    rows: Number(event.target.value),
                    page: '0'
                });
            }}
            onPageChange={(_, newPage) => {
                setState({ page: `${newPage}` });
            }}
            columns={[
                {
                    label: 'Created',
                    tooltip: 'Time the message was sent',
                    sortKey: 'created_on',
                    sortType: 'date',
                    sortable: false,
                    align: 'center',
                    width: 220,
                    hideSm: true,
                    renderer: (row) => (
                        <Typography variant={'body1'}>
                            {`${formatISO9075(row.created_on)}`}
                        </Typography>
                    )
                },
                {
                    label: 'Name',
                    tooltip: 'Persona Name',
                    sortKey: 'persona_name',
                    width: 250,
                    align: 'left',
                    renderer: (row) => (
                        <PersonCellFieldNonInteractive
                            steam_id={row.steam_id}
                            avatar_hash={
                                row.avatar_hash != ''
                                    ? row.avatar_hash
                                    : defaultAvatarHash
                            }
                            personaname={row.persona_name}
                        />
                    )
                },
                {
                    label: 'Message',
                    tooltip: 'Message',
                    sortKey: 'body',
                    align: 'left',
                    renderer: (row) => {
                        return (
                            <Grid container>
                                <Grid xs padding={1}>
                                    <Typography variant={'body1'}>
                                        {row.body}
                                    </Typography>
                                </Grid>

                                {row.auto_filter_flagged > 0 && (
                                    <Grid
                                        xs={'auto'}
                                        padding={1}
                                        display="flex"
                                        justifyContent="center"
                                        alignItems="center"
                                    >
                                        <>
                                            <FlagIcon
                                                color={'error'}
                                                fontSize="small"
                                            />
                                        </>
                                    </Grid>
                                )}
                                <Grid
                                    xs={'auto'}
                                    display="flex"
                                    justifyContent="center"
                                    alignItems="center"
                                >
                                    <ChatContextMenu
                                        flagged={row.auto_filter_flagged > 0}
                                        steamId={row.steam_id}
                                        person_message_id={
                                            row.person_message_id
                                        }
                                    />
                                </Grid>
                            </Grid>
                        );
                    }
                }
            ]}
        />
    );
};
