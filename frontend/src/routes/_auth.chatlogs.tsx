import ChatIcon from '@mui/icons-material/Chat';
import ClearIcon from '@mui/icons-material/Clear';
import FilterAltIcon from '@mui/icons-material/FilterAlt';
import RestartAltIcon from '@mui/icons-material/RestartAlt';
import SearchIcon from '@mui/icons-material/Search';
import Button from '@mui/material/Button';
import ButtonGroup from '@mui/material/ButtonGroup';
import FormControl from '@mui/material/FormControl';
import InputLabel from '@mui/material/InputLabel';
import MenuItem from '@mui/material/MenuItem';
import Select from '@mui/material/Select';
import TextField from '@mui/material/TextField';
import Typography from '@mui/material/Typography';
import Grid from '@mui/material/Unstable_Grid2';
import { useForm } from '@tanstack/react-form';
import { useQuery } from '@tanstack/react-query';
import { createFileRoute, useLoaderData, useNavigate } from '@tanstack/react-router';
import { createColumnHelper, getCoreRowModel, useReactTable } from '@tanstack/react-table';
import stc from 'string-to-color';
import { z } from 'zod';
import { apiGetMessages, apiGetServers, PersonMessage, ServerSimple } from '../api';
import { ContainerWithHeader } from '../component/ContainerWithHeader.tsx';
import { DataTable, HeadingCell } from '../component/DataTable.tsx';
import { LoadingPlaceholder } from '../component/LoadingPlaceholder.tsx';
import { PaginationInfinite } from '../component/PaginationInfinite.tsx';
import { PersonCell } from '../component/PersonCell.tsx';
import { commonTableSearchSchema } from '../util/table.ts';
import { renderDateTime } from '../util/text.tsx';

const chatlogsSchema = z.object({
    ...commonTableSearchSchema,
    sortColumn: z
        .enum(['person_message_id', 'steam_id', 'persona_name', 'server_name', 'server_id', 'team', 'created_on', 'pattern'])
        .catch('person_message_id'),
    server_id: z.number().catch(0),
    persona_name: z.string().catch(''),
    body: z.string().catch(''),
    steam_id: z.string().catch('')
});

type chatLogForm = {
    server_id: number;
    persona_name: string;
    body: string;
    steam_id: string;
};

export const Route = createFileRoute('/_auth/chatlogs')({
    component: ChatLogs,
    validateSearch: (search) => chatlogsSchema.parse(search),
    loader: async ({ context }) => {
        return {
            servers: await context.queryClient.ensureQueryData({
                queryKey: ['serversSimple'],
                queryFn: apiGetServers
            })
        };
    }
});

const columnHelper = createColumnHelper<PersonMessage>();

function ChatLogs() {
    const { body, persona_name, steam_id, server_id, page, sortColumn, rows, sortOrder } = Route.useSearch();
    //const { currentUser } = useCurrentUserCtx();
    const { servers } = useLoaderData({ from: '/_auth/chatlogs' }) as { servers: ServerSimple[] };
    const navigate = useNavigate({ from: Route.fullPath });

    const { data: logs, isLoading } = useQuery({
        queryKey: ['chatlogs', { page, server_id, persona_name, steam_id }],
        queryFn: async () => {
            return await apiGetMessages({
                server_id: server_id,
                personaname: persona_name,
                query: body,
                source_id: steam_id,
                limit: Number(rows),
                offset: Number(page ?? 0) * Number(rows),
                order_by: sortColumn,
                desc: sortOrder == 'desc'
            });
        }
    });

    const { Field, Subscribe, handleSubmit, reset } = useForm<chatLogForm>({
        onSubmit: async ({ value }) => {
            await navigate({ search: (prev) => ({ ...prev, ...value }) });
        },
        defaultValues: {
            body,
            persona_name,
            server_id,
            steam_id
        }
    });

    return (
        <>
            <Grid container spacing={2}>
                <Grid xs={12}>
                    <ContainerWithHeader title={'Chat Filters'} iconLeft={<FilterAltIcon />}>
                        <form
                            onSubmit={(e) => {
                                e.preventDefault();
                                e.stopPropagation();
                                handleSubmit();
                            }}
                        >
                            <Grid container padding={2} spacing={2} justifyContent={'center'} alignItems={'center'}>
                                <Grid xs={6} md={3}>
                                    <Field
                                        name={'persona_name'}
                                        children={({ state, handleChange, handleBlur }) => {
                                            return (
                                                <>
                                                    <TextField
                                                        fullWidth
                                                        label="Name"
                                                        defaultValue={state.value}
                                                        onChange={(e) => handleChange(e.target.value)}
                                                        onBlur={handleBlur}
                                                        variant="outlined"
                                                    />
                                                </>
                                            );
                                        }}
                                    />
                                </Grid>
                                <Grid xs={6} md={3}>
                                    <Field
                                        name={'steam_id'}
                                        children={({ state, handleChange, handleBlur }) => {
                                            return (
                                                <>
                                                    <TextField
                                                        fullWidth
                                                        label="SteamID"
                                                        value={state.value}
                                                        onChange={(e) => handleChange(e.target.value)}
                                                        onBlur={handleBlur}
                                                        variant="outlined"
                                                    />
                                                </>
                                            );
                                        }}
                                    />
                                </Grid>
                                <Grid xs={6} md={3}>
                                    <Field
                                        name={'body'}
                                        children={({ state, handleChange, handleBlur }) => {
                                            return (
                                                <>
                                                    <TextField
                                                        fullWidth
                                                        label="Message"
                                                        value={state.value}
                                                        onChange={(e) => handleChange(e.target.value)}
                                                        onBlur={handleBlur}
                                                        variant="outlined"
                                                    />
                                                </>
                                            );
                                        }}
                                    />
                                </Grid>
                                <Grid xs={6} md={3}>
                                    <Field
                                        name={'server_id'}
                                        children={({ state, handleChange, handleBlur }) => {
                                            return (
                                                <>
                                                    <FormControl fullWidth>
                                                        <InputLabel id="server-select-label">Servers</InputLabel>
                                                        <Select
                                                            fullWidth
                                                            value={state.value}
                                                            label="Servers"
                                                            onChange={(e) => {
                                                                handleChange(Number(e.target.value));
                                                            }}
                                                            onBlur={handleBlur}
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
                                <Grid xs={12} md={12}>
                                    <Subscribe
                                        selector={(state) => [state.canSubmit, state.isSubmitting]}
                                        children={([canSubmit, isSubmitting]) => (
                                            <ButtonGroup>
                                                <Button
                                                    type="submit"
                                                    disabled={!canSubmit}
                                                    variant={'contained'}
                                                    color={'success'}
                                                    startIcon={<SearchIcon />}
                                                >
                                                    {isSubmitting ? '...' : 'Search'}
                                                </Button>
                                                <Button
                                                    type="reset"
                                                    onClick={() => reset()}
                                                    variant={'contained'}
                                                    color={'warning'}
                                                    startIcon={<RestartAltIcon />}
                                                >
                                                    Reset
                                                </Button>
                                                <Button
                                                    type="button"
                                                    onClick={async () => {
                                                        await navigate({
                                                            search: (prev) => {
                                                                return {
                                                                    ...prev,
                                                                    page: 0,
                                                                    steam_id: '',
                                                                    body: '',
                                                                    persona_name: '',
                                                                    server_id: 0
                                                                };
                                                            }
                                                        });
                                                    }}
                                                    variant={'contained'}
                                                    color={'error'}
                                                    startIcon={<ClearIcon />}
                                                >
                                                    Clear
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
                    <ChatTable rows={isLoading || !logs ? [] : logs} servers={servers} isLoading={isLoading} />
                </Grid>
            </Grid>
        </>
    );
}

const ChatTable = ({ rows, servers, isLoading }: { rows: PersonMessage[]; servers: ServerSimple[]; isLoading: boolean }) => {
    const { page } = Route.useSearch();
    const navigate = useNavigate({ from: '/chatlogs' });

    const columns = [
        columnHelper.accessor('server_id', {
            header: () => <HeadingCell name={'Server'} />,
            cell: (info) => {
                const serv = servers.find((s) => (s.server_id = info.getValue())) || { server_name: 'unk-1' };
                return (
                    <Button
                        sx={{
                            color: stc(serv.server_name)
                        }}
                        onClick={async () => {
                            await navigate({ search: (prev) => ({ ...prev, server_id: info.getValue() }) });
                        }}
                    >
                        {serv?.server_name}
                    </Button>
                );
            },
            footer: () => <HeadingCell name={'Server'} />
        }),
        columnHelper.accessor('created_on', {
            header: () => <HeadingCell name={'Created'} />,
            cell: (info) => <Typography align={'center'}>{renderDateTime(info.getValue())}</Typography>,
            footer: () => <HeadingCell name={'Created'} />
        }),
        columnHelper.accessor('persona_name', {
            header: () => <HeadingCell name={'Name'} />,
            cell: (info) => (
                <PersonCell
                    steam_id={info.row.getValue('steam_id')}
                    avatar_hash={rows[info.row.index].avatar_hash}
                    personaname={info.row.getValue('persona_name')}
                    onClick={async () => {
                        await navigate({ params: { steamId: info.row.getValue<string>('steam_id') }, to: `/profile/$steamId` });
                    }}
                />
            ),
            footer: () => <HeadingCell name={'Name'} />
        }),
        columnHelper.accessor('body', {
            header: () => <HeadingCell name={'Message'} />,
            cell: (info) => (
                <Typography padding={0} variant={'body1'}>
                    {info.getValue()}
                </Typography>
            ),
            footer: () => <HeadingCell name={'Message'} />
        })
    ];

    const table = useReactTable({
        data: rows,
        columns: columns,
        getCoreRowModel: getCoreRowModel(),
        manualPagination: true,
        autoResetPageIndex: true
    });

    return (
        <ContainerWithHeader iconLeft={<ChatIcon />} title={'Chat Logs'}>
            {isLoading ? <LoadingPlaceholder /> : <DataTable table={table} />}
            <PaginationInfinite route={'/_auth/chatlogs'} page={page} />
        </ContainerWithHeader>
    );
};

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
