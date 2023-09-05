import Button from '@mui/material/Button';
import { steamIdQueryValue, stringHexNumber } from '../util/text';
import { format } from 'date-fns';
import IconButton from '@mui/material/IconButton';
import React, { useState } from 'react';
import MoreVertIcon from '@mui/icons-material/MoreVert';
import Menu from '@mui/material/Menu';
import MenuItem from '@mui/material/MenuItem';
import { DataTable, RowsPerPage } from './DataTable';
import { PersonMessage } from '../api';
import { MessageContextModal } from './MessageContextModal';
import { PersonCell } from './PersonCell';

export interface PersonMessageTableProps {
    messages: PersonMessage[];
    selectedIndex?: number;
}

export const PersonMessageTable = ({ messages }: PersonMessageTableProps) => {
    const [anchorEl, setAnchorEl] = React.useState<null | HTMLElement>(null);
    const [menuOpen, setMenuOpen] = React.useState<boolean>(false);
    const [selectedMessageId, setSelectedMessageId] = useState<number>(0);
    const [contextOpen, setContextOpen] = useState<boolean>(false);
    return (
        <>
            <MessageContextModal
                open={contextOpen}
                setOpen={setContextOpen}
                messageId={selectedMessageId}
            />
            <DataTable
                preSelectIndex={selectedMessageId}
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
                                    variant={'contained'}
                                    sx={{
                                        backgroundColor: stringHexNumber(
                                            row.server_name
                                        )
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
                        queryValue: (row) =>
                            row.persona_name + steamIdQueryValue(row.steam_id),
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
                        align: 'left',
                        queryValue: (row) => row.body
                    },
                    {
                        label: 'Act',
                        tooltip: 'Actions',
                        virtual: true,
                        virtualKey: 'action',
                        sortable: false,
                        align: 'right',
                        renderer: (row) => {
                            return (
                                <>
                                    <IconButton
                                        color={'secondary'}
                                        onClick={(
                                            event: React.MouseEvent<HTMLElement>
                                        ) => {
                                            setAnchorEl(event.currentTarget);
                                            setMenuOpen(true);
                                        }}
                                    >
                                        <MoreVertIcon />
                                    </IconButton>
                                    <Menu
                                        anchorEl={anchorEl}
                                        open={menuOpen}
                                        onClose={() => {
                                            setAnchorEl(null);
                                            setMenuOpen(false);
                                        }}
                                    >
                                        <MenuItem
                                            onClick={() => {
                                                setSelectedMessageId(
                                                    row.person_message_id
                                                );
                                                setContextOpen(true);
                                                setMenuOpen(false);
                                                setAnchorEl(null);
                                            }}
                                        >
                                            Context
                                        </MenuItem>
                                    </Menu>
                                </>
                            );
                        }
                    }
                ]}
                defaultSortColumn={'created_on'}
                rowsPerPage={RowsPerPage.TwentyFive}
                rows={messages}
            />
        </>
    );
};
