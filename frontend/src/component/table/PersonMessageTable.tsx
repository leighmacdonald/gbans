import { ChangeEvent, useEffect, useState } from 'react';
import Button from '@mui/material/Button';
import { format } from 'date-fns';
import stc from 'string-to-color';
import { apiGetMessages, PersonMessage } from '../../api';
import { logErr } from '../../util/errors';
import { PersonCell } from '../PersonCell';
import { LazyTable, Order, RowsPerPage } from './LazyTable';

export interface PersonMessageTableProps {
    steam_id: string;
    selectedIndex?: number;
}

export const PersonMessageTable = ({ steam_id }: PersonMessageTableProps) => {
    const [messages, setMessages] = useState<PersonMessage[]>([]);
    const [sortOrder, setSortOrder] = useState<Order>('desc');
    const [sortColumn, setSortColumn] =
        useState<keyof PersonMessage>('person_message_id');
    const [rowPerPageCount, setRowPerPageCount] = useState<number>(
        RowsPerPage.Ten
    );
    const [loading, setLoading] = useState(false);
    const [page, setPage] = useState(0);
    const [totalRows, setTotalRows] = useState<number>(0);

    useEffect(() => {
        const abortController = new AbortController();
        setLoading(true);
        apiGetMessages(
            {
                limit: rowPerPageCount,
                offset: page * rowPerPageCount,
                order_by: sortColumn,
                desc: sortOrder == 'desc',
                source_id: steam_id
            },
            abortController
        )
            .then((resp) => {
                setMessages(resp.data);
                setTotalRows(resp.count);
            })
            .catch((e) => {
                logErr(e);
            })
            .finally(() => {
                setLoading(false);
            });
        return () => abortController.abort();
    }, [page, rowPerPageCount, sortColumn, sortOrder, steam_id]);

    return (
        <LazyTable
            loading={loading}
            showPager={true}
            count={totalRows}
            rows={messages}
            page={page}
            rowsPerPage={rowPerPageCount}
            sortOrder={sortOrder}
            sortColumn={sortColumn}
            onSortColumnChanged={async (column) => {
                setSortColumn(column);
            }}
            onSortOrderChanged={async (direction) => {
                setSortOrder(direction);
            }}
            onPageChange={(_, newPage: number) => {
                setPage(newPage);
            }}
            onRowsPerPageChange={(
                event: ChangeEvent<HTMLInputElement | HTMLTextAreaElement>
            ) => {
                setRowPerPageCount(parseInt(event.target.value, 10));
                setPage(0);
            }}
            columns={[
                {
                    label: 'Server',
                    tooltip: 'Server',
                    sortKey: 'server_name',
                    sortType: 'string',
                    align: 'left',
                    width: '50px',
                    renderer: (row) => {
                        return (
                            <Button
                                fullWidth
                                variant={'text'}
                                sx={{
                                    color: stc(row.server_name)
                                }}
                            >
                                {row.server_name}
                            </Button>
                        );
                    }
                },
                {
                    label: 'Created',
                    tooltip: 'Created On',
                    sortKey: 'created_on',
                    sortType: 'date',
                    align: 'left',
                    width: '120px',
                    renderer: (row) => {
                        return format(row.created_on, 'dd-MMM HH:mm');
                    }
                },
                {
                    label: 'Name',
                    tooltip: 'Name',
                    sortKey: 'persona_name',
                    sortType: 'string',
                    align: 'left',
                    width: '150px',
                    renderer: (row) => (
                        <PersonCell
                            steam_id={row.steam_id}
                            personaname={row.persona_name}
                            avatar_hash={
                                'fef49e7fa7e1997310d705b2a6158ff8dc1cdfeb'
                            }
                        ></PersonCell>
                    )
                },
                {
                    label: 'Message',
                    tooltip: 'Message',
                    sortKey: 'body',
                    sortType: 'string',
                    align: 'left'
                }
                // {
                //     label: 'Act',
                //     tooltip: 'Actions',
                //     virtual: true,
                //     virtualKey: 'action',
                //     sortable: false,
                //     align: 'right',
                //     renderer: (row) => {
                //         return (
                //             <>
                //                 <IconButton
                //                     color={'secondary'}
                //                     onClick={(
                //                         event: MouseEvent<HTMLElement>
                //                     ) => {
                //                         setAnchorEl(event.currentTarget);
                //                         setMenuOpen(true);
                //                     }}
                //                 >
                //                     <MoreVertIcon />
                //                 </IconButton>
                //                 <Menu
                //                     anchorEl={anchorEl}
                //                     open={menuOpen}
                //                     onClose={() => {
                //                         setAnchorEl(null);
                //                         setMenuOpen(false);
                //                     }}
                //                 >
                //                     <MenuItem
                //                         onClick={async () => {
                //                             await NiceModal.show(
                //                                 ModalMessageContext,
                //                                 {
                //                                     messageId:
                //                                         row.person_message_id
                //                                 }
                //                             );
                //                         }}
                //                     >
                //                         Context
                //                     </MenuItem>
                //                 </Menu>
                //             </>
                //         );
                //     }
                // }
            ]}
        />
    );
};
