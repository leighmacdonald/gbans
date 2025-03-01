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
import { Title } from '../component/Title.tsx';
import { Buttons } from '../component/field/Buttons.tsx';
import { SteamIDField } from '../component/field/SteamIDField.tsx';
import { TextFieldSimple } from '../component/field/TextFieldSimple.tsx';
import { ensureFeatureEnabled } from '../util/features.ts';
import { makeCommonTableSearchSchema, RowsPerPage } from '../util/table.ts';

const chatlogsSchema = z.object({
    ...makeCommonTableSearchSchema([
        'person_message_id',
        'steam_id',
        'persona_name',
        'server_name',
        'server_id',
        'team',
        'created_on',
        'pattern',
        'auto_filter_flagged'
    ]),
    server_id: z.number().optional(),
    persona_name: z.string().optional(),
    body: z.string().optional(),
    steam_id: z.string().optional(),
    flagged_only: z.boolean().optional(),
    autoRefresh: z.number().optional()
});

export const Route = createFileRoute('/_auth/chatlogs')({
    component: ChatLogs,
    beforeLoad: () => {
        ensureFeatureEnabled('chatlogs_enabled');
    },
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
    const search = Route.useSearch();
    const { hasPermission } = useRouteContext({ from: '/_auth/chatlogs' });
    const { servers } = useLoaderData({ from: '/_auth/chatlogs' }) as { servers: ServerSimple[] };
    const navigate = useNavigate({ from: Route.fullPath });

    const { data: messages, isLoading } = useQuery({
        queryKey: ['chatlogs', { search }],
        queryFn: async () => {
            return await apiGetMessages({
                server_id: search.server_id,
                personaname: search.persona_name,
                query: search.body,
                source_id: search.steam_id,
                limit: search.pageSize ?? defaultRows,
                offset: (search.pageIndex ?? 0) * (search.pageSize ?? defaultRows),
                order_by: 'person_message_id',
                desc: (search.sortOrder ?? 'desc') == 'desc',
                flagged_only: search.flagged_only ?? false
            });
        },
        refetchInterval: search.autoRefresh
    });

    const { Field, Subscribe, handleSubmit, reset } = useForm({
        onSubmit: async ({ value }) => {
            await navigate({ to: '/chatlogs', search: (prev) => ({ ...prev, ...value }) });
        },
        defaultValues: {
            body: search.body ?? '',
            persona_name: search.persona_name ?? '',
            server_id: search.server_id ?? 0,
            steam_id: search.steam_id ?? '',
            flagged_only: search.flagged_only ?? false,
            autoRefresh: search.autoRefresh ?? 0
        }
    });

    const clear = async () => {
        reset();
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
        await handleSubmit();
    };
    return (
        <>
            <Title>Chat Logs</Title>

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
                                        children={(props) => {
                                            return <SteamIDField {...props} fullwidth={true} />;
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
                            page={search.pageIndex ?? 0}
                            rows={search.pageSize ?? defaultRows}
                            path={'/chatlogs'}
                            data={{ data: [], count: -1 }}
                        />
                    </ContainerWithHeader>
                </Grid>
            </Grid>
        </>
    );
}
