import ChatIcon from '@mui/icons-material/Chat';
import FilterAltIcon from '@mui/icons-material/FilterAlt';
import Checkbox from '@mui/material/Checkbox';
import FormControl from '@mui/material/FormControl';
import FormControlLabel from '@mui/material/FormControlLabel';
import FormGroup from '@mui/material/FormGroup';
import InputLabel from '@mui/material/InputLabel';
import MenuItem from '@mui/material/MenuItem';
import Select from '@mui/material/Select';
import Grid from '@mui/material/Unstable_Grid2';
import { useForm } from '@tanstack/react-form';
import { useQuery } from '@tanstack/react-query';
import { createFileRoute, useLoaderData, useNavigate, useRouteContext } from '@tanstack/react-router';
import { z } from 'zod';
import { apiGetMessages, apiGetServers, PermissionLevel, ServerSimple } from '../api';
import { ChatTable } from '../component/ChatTable.tsx';
import { ContainerWithHeader } from '../component/ContainerWithHeader.tsx';
import { Paginator } from '../component/Paginator.tsx';
import { Buttons } from '../component/field/Buttons.tsx';
import { SteamIDField } from '../component/field/SteamIDField.tsx';
import { TextFieldSimple } from '../component/field/TextFieldSimple.tsx';
import { commonTableSearchSchema, RowsPerPage } from '../util/table.ts';

const chatlogsSchema = z.object({
    ...commonTableSearchSchema,
    sortColumn: z
        .enum([
            'person_message_id',
            'steam_id',
            'persona_name',
            'server_name',
            'server_id',
            'team',
            'created_on',
            'pattern',
            'auto_filter_flagged'
        ])
        .optional(),
    server_id: z.number().optional(),
    persona_name: z.string().optional(),
    body: z.string().optional(),
    steam_id: z.string().optional(),
    flagged_only: z.boolean().optional(),
    autoRefresh: z.number().optional()
});

type chatLogForm = {
    server_id: number;
    persona_name: string;
    body: string;
    steam_id: string;
    flagged_only: boolean;
    autoRefresh: number;
};

export const Route = createFileRoute('/_auth/chatlogs')({
    component: ChatLogs,
    validateSearch: (search) => chatlogsSchema.parse(search),
    loader: async ({ context }) => {
        const unsorted = await context.queryClient.ensureQueryData({
            queryKey: ['serversSimple'],
            queryFn: apiGetServers
        });
        return {
            servers: unsorted.sort((a, b) => {
                if (a.server_name > b.server_name) {
                    return 1;
                }
                if (a.server_name < b.server_name) {
                    return -1;
                }
                return 0;
            })
        };
    }
});

function ChatLogs() {
    const defaultRows = RowsPerPage.TwentyFive;
    const { body, autoRefresh, persona_name, steam_id, server_id, page, sortColumn, flagged_only, rows, sortOrder } =
        Route.useSearch();
    //const { currentUser } = useCurrentUserCtx();
    const { hasPermission } = useRouteContext({ from: '/_auth/chatlogs' });
    const { servers } = useLoaderData({ from: '/_auth/chatlogs' }) as { servers: ServerSimple[] };
    const navigate = useNavigate({ from: Route.fullPath });

    const { data: messages, isLoading } = useQuery({
        queryKey: [
            'chatlogs',
            { page, server_id, persona_name, steam_id, rows, sortOrder, sortColumn, body, autoRefresh, flagged_only }
        ],
        queryFn: async () => {
            return await apiGetMessages({
                server_id: server_id,
                personaname: persona_name,
                query: body,
                source_id: steam_id,
                limit: rows ?? defaultRows,
                offset: (page ?? 0) * (rows ?? defaultRows),
                order_by: sortColumn,
                desc: (sortOrder ?? 'desc') == 'desc',
                flagged_only: flagged_only ?? false
            });
        },
        refetchInterval: autoRefresh
    });

    const { Field, Subscribe, handleSubmit, reset } = useForm<chatLogForm>({
        onSubmit: async ({ value }) => {
            await navigate({ to: '/chatlogs', search: (prev) => ({ ...prev, ...value }) });
        },
        defaultValues: {
            body: body ?? '',
            persona_name: persona_name ?? '',
            server_id: server_id ?? 0,
            steam_id: steam_id ?? '',
            flagged_only: flagged_only ?? false,
            autoRefresh: autoRefresh ?? 0
        }
    });

    const clear = async () => {
        await navigate({
            to: '/chatlogs',
            search: (prev) => ({
                ...prev,
                body: undefined,
                persona_name: undefined,
                server_id: undefined,
                steam_id: undefined,
                flagged_only: undefined,
                autoRefresh: undefined
            })
        });
    };
    return (
        <>
            <Grid container spacing={2}>
                <Grid xs={12}>
                    <ContainerWithHeader title={'Chat Filters'} iconLeft={<FilterAltIcon />}>
                        <form
                            onSubmit={async (e) => {
                                e.preventDefault();
                                e.stopPropagation();
                                await handleSubmit();
                            }}
                        >
                            <Grid container padding={2} spacing={2} justifyContent={'center'} alignItems={'center'}>
                                <Grid xs={6} md={3}>
                                    <Field
                                        name={'persona_name'}
                                        children={(props) => {
                                            return <TextFieldSimple {...props} label={'Name'} />;
                                        }}
                                    />
                                </Grid>
                                <Grid xs={6} md={3}>
                                    <Field
                                        name={'steam_id'}
                                        children={({ state, handleChange, handleBlur }) => {
                                            return (
                                                <SteamIDField
                                                    state={state}
                                                    handleBlur={handleBlur}
                                                    handleChange={handleChange}
                                                    fullwidth={true}
                                                />
                                            );
                                        }}
                                    />
                                </Grid>
                                <Grid xs={6} md={3}>
                                    <Field
                                        name={'body'}
                                        children={(props) => {
                                            return <TextFieldSimple {...props} label={'Message'} />;
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
                                                            {servers.map((s) => (
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
                                {hasPermission(PermissionLevel.Moderator) && (
                                    <>
                                        <Grid xs={'auto'}>
                                            <Field
                                                name={'flagged_only'}
                                                children={({ state, handleChange, handleBlur }) => {
                                                    return (
                                                        <>
                                                            <FormGroup>
                                                                <FormControlLabel
                                                                    control={
                                                                        <Checkbox
                                                                            checked={state.value}
                                                                            onBlur={handleBlur}
                                                                            onChange={(_, v) => {
                                                                                handleChange(v);
                                                                            }}
                                                                        />
                                                                    }
                                                                    label="Flagged Only"
                                                                />
                                                            </FormGroup>
                                                        </>
                                                    );
                                                }}
                                            />
                                        </Grid>
                                        <Grid xs>
                                            <Field
                                                name={'autoRefresh'}
                                                children={({ state, handleChange, handleBlur }) => {
                                                    return (
                                                        <FormControl fullWidth>
                                                            <InputLabel id="server-select-label">
                                                                Auto-Refresh
                                                            </InputLabel>
                                                            <Select
                                                                fullWidth
                                                                value={state.value}
                                                                label="Auto Refresh"
                                                                onChange={(e) => {
                                                                    handleChange(Number(e.target.value));
                                                                }}
                                                                onBlur={handleBlur}
                                                            >
                                                                <MenuItem value={0}>Off</MenuItem>
                                                                <MenuItem value={10000}>10sec</MenuItem>
                                                                <MenuItem value={30000}>30sec</MenuItem>
                                                                <MenuItem value={60000}>1min</MenuItem>
                                                                <MenuItem value={300000}>5min</MenuItem>
                                                            </Select>
                                                        </FormControl>
                                                    );
                                                }}
                                            />
                                        </Grid>
                                    </>
                                )}

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
                    <ContainerWithHeader iconLeft={<ChatIcon />} title={'Chat Logs'}>
                        <ChatTable messages={messages ?? []} isLoading={isLoading} />
                        <Paginator
                            page={page ?? 0}
                            rows={rows ?? defaultRows}
                            path={'/chatlogs'}
                            data={{ data: [], count: -1 }}
                        />
                    </ContainerWithHeader>
                </Grid>
            </Grid>
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
