import ChatIcon from '@mui/icons-material/Chat';
import FilterAltIcon from '@mui/icons-material/FilterAlt';
import Button from '@mui/material/Button';
import ButtonGroup from '@mui/material/ButtonGroup';
import FormControl from '@mui/material/FormControl';
import InputLabel from '@mui/material/InputLabel';
import MenuItem from '@mui/material/MenuItem';
import Select from '@mui/material/Select';
import Table from '@mui/material/Table';
import TableBody from '@mui/material/TableBody';
import TableCell from '@mui/material/TableCell';
import TableContainer from '@mui/material/TableContainer';
import TableHead from '@mui/material/TableHead';
import TableRow from '@mui/material/TableRow';
import TextField from '@mui/material/TextField';
import Typography from '@mui/material/Typography';
import Grid from '@mui/material/Unstable_Grid2';
import { FieldApi, useForm } from '@tanstack/react-form';
import { useQuery } from '@tanstack/react-query';
import { createFileRoute, useNavigate } from '@tanstack/react-router';
import { createColumnHelper, flexRender, getCoreRowModel, useReactTable } from '@tanstack/react-table';
import { z } from 'zod';
import { apiGetMessages, apiGetServers, PersonMessage } from '../api';
import { ContainerWithHeader } from '../component/ContainerWithHeader';
import { RowsPerPage } from '../util/table.ts';

const chatlogsSchema = z.object({
    sortOrder: z.enum(['desc', 'asc']).catch('desc'),
    sortColumn: z
        .enum(['person_message_id', 'steam_id', 'persona_name', 'server_name', 'server_id', 'team', 'created_on', 'pattern'])
        .catch('person_message_id'),
    page: z.number().min(0).catch(0),
    rowPerPageCount: z.number().min(RowsPerPage.Ten).max(RowsPerPage.Hundred).catch(RowsPerPage.TwentyFive),
    server: z.number().catch(0),
    personaName: z.string().catch(''),
    message: z.string().catch(''),
    steamId: z.string().catch('')
});

type chatLogForm = {
    server: number;
    personaName: string;
    message: string;
    steamId: string;
};

export const Route = createFileRoute('/chatlogs')({
    component: ChatLogs,
    validateSearch: (search) => chatlogsSchema.parse(search)
});

const columnHelper = createColumnHelper<PersonMessage>();

const columns = [
    columnHelper.accessor('server_id', {
        cell: (info) => info.getValue(),
        footer: (props) => props.column.id
    }),
    columnHelper.accessor('persona_name', {
        cell: (info) => info.getValue(),
        footer: (props) => props.column.id
    }),
    columnHelper.accessor('body', {
        cell: (info) => info.getValue(),
        footer: (props) => props.column.id
    })
];

function ChatLogs() {
    const { message, personaName, steamId, server, page, sortColumn, rowPerPageCount, sortOrder } = Route.useSearch();
    //const [totalRows, setTotalRows] = useState<number>(0);
    //const { currentUser } = useCurrentUserCtx();
    const navigate = useNavigate({ from: Route.fullPath });

    const { data: servers, isLoading: isLoadingServers } = useQuery({
        queryKey: ['serversSimple'],
        queryFn: apiGetServers
    });

    const { data: rows } = useQuery({
        queryKey: ['chatlogs'],
        queryFn: async () => {
            console.log('running query');
            const resp = await apiGetMessages({
                server_id: server,
                personaname: personaName,
                query: message,
                source_id: steamId,
                limit: Number(rowPerPageCount),
                offset: Number(page ?? 0) * Number(rowPerPageCount),
                order_by: sortColumn,
                desc: sortOrder == 'desc'
            });
            return resp.data;
        }
    });

    const form = useForm<chatLogForm>({
        onSubmit: async ({ value }) => {
            console.log(value);
            await navigate({
                replace: true,
                search: (prev) => ({
                    ...prev,
                    message: value.message,
                    personaName: value.personaName,
                    steamId: value.steamId,
                    server: value.server
                })
            });
        },
        defaultValues: {
            message: message,
            personaName: personaName,
            server: server,
            steamId: steamId
        }
    });

    const table = useReactTable({
        data: rows ?? [],
        columns: columns,
        getCoreRowModel: getCoreRowModel()
    });

    //const { data: realServers } = useServers();

    return (
        <>
            <Grid container spacing={2}>
                <Grid xs={12}>
                    <ContainerWithHeader title={'Chat Filters'} iconLeft={<FilterAltIcon />}>
                        <form
                            onSubmit={async (e) => {
                                e.preventDefault();
                                e.stopPropagation();
                                await form.handleSubmit();
                            }}
                        >
                            <Grid container padding={2} spacing={2} justifyContent={'center'} alignItems={'center'}>
                                <Grid xs={6} md={3}>
                                    <form.Field
                                        name={'personaName'}
                                        children={(field) => {
                                            return (
                                                <>
                                                    <TextField
                                                        fullWidth
                                                        id={field.name}
                                                        name={field.name}
                                                        label="Name"
                                                        value={field.state.value}
                                                        onChange={(e) => field.handleChange(e.target.value)}
                                                        onBlur={field.handleBlur}
                                                        variant="outlined"
                                                    />
                                                    <FieldInfo field={field} />
                                                </>
                                            );
                                        }}
                                    />
                                </Grid>
                                <Grid xs={6} md={3}>
                                    <form.Field
                                        name={'steamId'}
                                        children={(field) => {
                                            return (
                                                <>
                                                    <TextField
                                                        fullWidth
                                                        id={field.name}
                                                        name={field.name}
                                                        label="SteamID"
                                                        value={field.state.value}
                                                        onChange={(e) => field.handleChange(e.target.value)}
                                                        onBlur={field.handleBlur}
                                                        variant="outlined"
                                                    />
                                                    <FieldInfo field={field} />
                                                </>
                                            );
                                        }}
                                    />
                                </Grid>
                                <Grid xs={6} md={3}>
                                    <form.Field
                                        name={'message'}
                                        children={(field) => {
                                            return (
                                                <>
                                                    <TextField
                                                        fullWidth
                                                        id={field.name}
                                                        name={field.name}
                                                        label="Message"
                                                        value={field.state.value}
                                                        onChange={(e) => field.handleChange(e.target.value)}
                                                        onBlur={field.handleBlur}
                                                        variant="outlined"
                                                    />
                                                    <FieldInfo field={field} />
                                                </>
                                            );
                                        }}
                                    />
                                </Grid>
                                <Grid xs={6} md={3}>
                                    <form.Field
                                        name={'server'}
                                        children={(field) => {
                                            return (
                                                <>
                                                    <FormControl fullWidth>
                                                        <InputLabel id="server-select-label">Servers</InputLabel>
                                                        <Select<number>
                                                            fullWidth
                                                            labelId="server_ids-label"
                                                            id={field.name}
                                                            disabled={isLoadingServers}
                                                            value={field.state.value}
                                                            name={field.name}
                                                            label="Servers"
                                                            onChange={(e) => {
                                                                console.log(e.target);
                                                                field.handleChange(Number(e.target.value));
                                                            }}
                                                            onBlur={field.handleBlur}
                                                        >
                                                            <MenuItem value={0}>All</MenuItem>
                                                            {servers &&
                                                                servers.map((s) => (
                                                                    <MenuItem value={s.server_id} key={s.server_id}>
                                                                        {s.server_name}
                                                                    </MenuItem>
                                                                ))}
                                                        </Select>
                                                    </FormControl>
                                                    <FieldInfo field={field} />
                                                </>
                                            );
                                        }}
                                    />
                                </Grid>
                                {/*<Grid xs={3}>*/}
                                {/*    {currentUser.permission_level >=*/}
                                {/*        PermissionLevel.Moderator && (*/}
                                {/*        <AutoRefreshField />*/}
                                {/*    )}*/}
                                {/*</Grid>*/}
                                <Grid xs={6} md={3}>
                                    <form.Subscribe
                                        selector={(state) => [state.canSubmit, state.isSubmitting]}
                                        children={([canSubmit, isSubmitting]) => (
                                            <ButtonGroup>
                                                <Button type="submit" disabled={!canSubmit} variant={'contained'}>
                                                    {isSubmitting ? '...' : 'Submit'}
                                                </Button>
                                                <Button type="reset" onClick={() => form.reset()} variant={'contained'}>
                                                    Reset
                                                </Button>
                                            </ButtonGroup>
                                        )}
                                    />
                                </Grid>
                            </Grid>
                        </form>
                    </ContainerWithHeader>
                </Grid>
                <Grid xs={12}>
                    <ContainerWithHeader iconLeft={<ChatIcon />} title={'Chat Logs'}>
                        <TableContainer>
                            <Table>
                                <TableHead>
                                    {table.getHeaderGroups().map((headerGroup) => (
                                        <TableRow key={headerGroup.id}>
                                            {headerGroup.headers.map((header) => (
                                                <TableCell key={header.id}>
                                                    <Typography
                                                        padding={0}
                                                        sx={{
                                                            fontWeight: 'bold'
                                                        }}
                                                        variant={'button'}
                                                    >
                                                        {header.isPlaceholder
                                                            ? null
                                                            : flexRender(header.column.columnDef.header, header.getContext())}
                                                    </Typography>
                                                </TableCell>
                                            ))}
                                        </TableRow>
                                    ))}
                                </TableHead>
                                <TableBody>
                                    {table.getRowModel().rows.map((row) => (
                                        <tr key={row.id}>
                                            {row.getVisibleCells().map((cell) => (
                                                <td key={cell.id}>{flexRender(cell.column.columnDef.cell, cell.getContext())}</td>
                                            ))}
                                        </tr>
                                    ))}
                                </TableBody>
                            </Table>
                        </TableContainer>
                    </ContainerWithHeader>
                </Grid>
            </Grid>
        </>
    );
}

function FieldInfo({ field }: { field: FieldApi<any, any, any, any> }) {
    return (
        <>
            {field.state.meta.touchedErrors ? <em>{field.state.meta.touchedErrors}</em> : null}
            {field.state.meta.isValidating ? 'Validating...' : null}
        </>
    );
}

//
// interface ChatContextMenuProps {
//     person_message_id: number;
//     flagged: boolean;
//     steamId: string;
// }
//
// const ChatContextMenu = ({
//                              person_message_id,
//                              flagged,
//                              steamId
//                          }: ChatContextMenuProps) => {
//     const navigate = useNavigate();
//
//     const [anchorEl, setAnchorEl] = useState<null | HTMLElement>(null);
//     const open = Boolean(anchorEl);
//     const handleClick = (event: MouseEvent<HTMLElement>) => {
//         setAnchorEl(event.currentTarget);
//     };
//     const handleClose = () => {
//         setAnchorEl(null);
//     };
//
//     const onClickReport = async () => {
//         sessionStorage.setItem(
//             sessionKeyReportPersonMessageIdName,
//             `${person_message_id}`
//         );
//         sessionStorage.setItem(sessionKeyReportSteamID, steamId);
//         await navigate({ to: '/report/' });
//         handleClose();
//     };
//
//     return (
//         <>
//             <IconButton onClick={handleClick} size={'small'}>
//                 <SettingsSuggestIcon color={'info'} />
//             </IconButton>
//             <Menu
//                 id='chat-msg-menu'
//                 anchorEl={anchorEl}
//                 open={open}
//                 onClose={handleClose}
//                 anchorOrigin={{
//                     vertical: 'top',
//                     horizontal: 'left'
//                 }}
//                 transformOrigin={{
//                     vertical: 'top',
//                     horizontal: 'left'
//                 }}
//             >
//                 <MenuItem onClick={onClickReport} disabled={flagged}>
//                     <ListItemIcon>
//                         <ReportIcon fontSize='small' color={'error'} />
//                     </ListItemIcon>
//                     <ListItemText>Create Report (Full)</ListItemText>
//                 </MenuItem>
//                 <MenuItem onClick={onClickReport} disabled={true}>
//                     <ListItemIcon>
//                         <ReportGmailerrorredIcon
//                             fontSize='small'
//                             color={'error'}
//                         />
//                     </ListItemIcon>
//                     <ListItemText>Create Report (1-Click)</ListItemText>
//                 </MenuItem>
//                 <Divider />
//                 <MenuItem onClick={handleClose} disabled={true}>
//                     <ListItemIcon>
//                         <HistoryIcon fontSize='small' />
//                     </ListItemIcon>
//                     <ListItemText>Message Context</ListItemText>
//                 </MenuItem>
//             </Menu>
//         </>
//     );
// };
