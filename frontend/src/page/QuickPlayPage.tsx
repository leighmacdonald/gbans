// import React, { useCallback, useEffect, useMemo, useState } from 'react';
// import { Heading } from '../component/Heading';
// import Paper from '@mui/material/Paper';
// import Grid from '@mui/material/Grid';
// import { DataTable, RowsPerPage } from '../component/DataTable';
// import {
//     apiServerQuery,
//     filterServerGameTypes,
//     qpAutoQueueMode,
//     qpGameType,
//     qpLobby,
//     qpMsgJoinedLobbySuccess,
//     qpMsgJoinLobbyRequest,
//     qpMsgLeaveLobbyRequest,
//     qpMsgType,
//     qpRequestTypes,
//     qpUserMessage,
//     qpUserMessageI,
//     SlimServer,
//     UserProfile
// } from '../api';
// import Button from '@mui/material/Button';
// import Link from '@mui/material/Link';
// import Stack from '@mui/material/Stack';
// import Switch from '@mui/material/Switch';
// import TextField from '@mui/material/TextField';
// import FormGroup from '@mui/material/FormGroup';
// import GroupAddIcon from '@mui/icons-material/GroupAdd';
// import GroupRemoveIcon from '@mui/icons-material/GroupRemove';
// import FormControlLabel from '@mui/material/FormControlLabel';
// import useWebSocket, { ReadyState } from 'react-use-websocket';
// import Typography from '@mui/material/Typography';
// import EmojiEventsIcon from '@mui/icons-material/EmojiEvents';
// import ButtonGroup from '@mui/material/ButtonGroup';
// import IconButton from '@mui/material/IconButton';
// import ChevronRightIcon from '@mui/icons-material/ChevronRight';
// import Tooltip from '@mui/material/Tooltip';
// import {
//     ConfirmationModal,
//     ConfirmationModalProps
// } from '../component/ConfirmationModal';
// import { useUserFlashCtx } from '../contexts/UserFlashCtx';
// import { ListItem, Select } from '@mui/material';
// import ListItemIcon from '@mui/material/ListItemIcon';
// import ListItemText from '@mui/material/ListItemText';
// import List from '@mui/material/List';
// import ListItemButton from '@mui/material/ListItemButton';
// import FormControl from '@mui/material/FormControl';
// import InputLabel from '@mui/material/InputLabel';
// import MenuItem from '@mui/material/MenuItem';
// import { GameModeSelect } from '../component/GameModeSelect';
//
// export type JoinLobbyModalProps = ConfirmationModalProps<string>;
//
// export const JoinLobbyModal = ({
//     open,
//     setOpen,
//     onSuccess
// }: JoinLobbyModalProps) => {
//     const [lobbyId, setLobbyId] = useState<string>('');
//     const { sendFlash } = useUserFlashCtx();
//
//     const handleSubmit = useCallback(() => {
//         if (lobbyId.length != 6) {
//             sendFlash('error', 'Invalid lobby ID');
//             return;
//         }
//         onSuccess && onSuccess(lobbyId);
//         setOpen(false);
//     }, [lobbyId, onSuccess, sendFlash, setOpen]);
//
//     return (
//         <ConfirmationModal
//             open={open}
//             setOpen={setOpen}
//             onSuccess={() => {
//                 setOpen(false);
//             }}
//             onCancel={() => {
//                 setOpen(false);
//             }}
//             onAccept={() => {
//                 handleSubmit();
//             }}
//             aria-labelledby="modal-title"
//             aria-describedby="modal-description"
//         >
//             <Stack spacing={2}>
//                 <Heading>Join a Lobby By ID</Heading>
//                 <Stack spacing={3} alignItems={'center'}>
//                     <TextField
//                         fullWidth
//                         label={'Lobby ID'}
//                         id={'lobbyId'}
//                         value={lobbyId}
//                         onChange={(evt) => {
//                             setLobbyId(evt.target.value);
//                         }}
//                     />
//                 </Stack>
//             </Stack>
//         </ConfirmationModal>
//     );
// };
//
// export const QuickPlayPage = () => {
//     const maxLobbyMembers = 6;
//     const [allServers, setAllServers] = useState<SlimServer[]>([]);
//     const [minPlayers, setMinPlayers] = useState<number>(0);
//     const [maxPlayers, setMaxPlayers] = useState<number>(0);
//     const [notFull, setNotFull] = useState<boolean>(true);
//     const [enableAutoQueue, setEnableAutoQueue] = useState<boolean>(false);
//     const [enableGameTypes, setEnableGameTypes] = useState<qpGameType[]>([]);
//     const [queueMode, setQueueMode] = useState<qpAutoQueueMode>('eager');
//     const [chatInput, setChatInput] = useState<string>('');
//     const [joinModalOpen, setJoinModalOpen] = useState(false);
//     const [lobby, setLobby] = useState<qpLobby>({ lobby_id: '', clients: [] });
//     const [messageHistory, setMessageHistory] = useState<qpUserMessageI[]>([]);
//
//     const token = useMemo(() => {
//         return localStorage.getItem('token') ?? '';
//     }, []);
//
//     const socketUrl = useMemo(() => {
//         const parsedUrl = new URL(window.location.href);
//         return `${parsedUrl.protocol == 'https' ? 'wss' : 'ws'}://${
//             parsedUrl.host
//         }/ws/quickplay`;
//     }, []);
//
//     const { readyState, lastJsonMessage, sendJsonMessage } = useWebSocket(
//         socketUrl,
//         {
//             onError: (event: WebSocketEventMap['error']) => {
//                 setMessageHistory((prevState) =>
//                     prevState.concat({
//                         message: event.type,
//                         created_at: new Date().toISOString()
//                     })
//                 );
//             },
//             onClose: () => {
//                 setMessageHistory((prevState) =>
//                     prevState.concat({
//                         message: 'Lobby connection closed',
//                         created_at: new Date().toISOString()
//                     })
//                 );
//             },
//             queryParams: {
//                 token: token
//             },
//             onOpen: () => {
//                 setMessageHistory((prevState) =>
//                     prevState.concat({
//                         message: 'Lobby connection opened',
//                         created_at: new Date().toISOString()
//                     })
//                 );
//             }, //Will attempt to reconnect on all close events, such as server shutting down
//             shouldReconnect: () => true
//         }
//     );
//
//     const isReady = useMemo(() => {
//         return readyState == ReadyState.OPEN;
//     }, [readyState]);
//
//     const sendMessage = useCallback(() => {
//         if (chatInput == '') {
//             return;
//         }
//         const req: qpUserMessage = {
//             msg_type: qpMsgType.qpMsgTypeSendMsgRequest,
//             payload: {
//                 message: chatInput,
//                 created_at: new Date().toISOString()
//             }
//         };
//         sendJsonMessage(req);
//         setChatInput('');
//     }, [chatInput, sendJsonMessage]);
//
//     const sendJoinLobbyRequest = useCallback(
//         (lobbyId: string) => {
//             if (lobbyId.length != 6) {
//                 return;
//             }
//             const request: qpMsgJoinLobbyRequest = {
//                 msg_type: qpMsgType.qpMsgTypeJoinLobbyRequest,
//                 payload: {
//                     lobby_id: lobbyId
//                 }
//             };
//             sendJsonMessage(request);
//         },
//         [sendJsonMessage]
//     );
//
//     const sendLeaveLobbyRequest = useCallback(() => {
//         if (!lobby) {
//             return;
//         }
//         const request: qpMsgLeaveLobbyRequest = {
//             msg_type: qpMsgType.qpMsgTypeLeaveLobbyRequest,
//             payload: {
//                 lobby_id: lobby.lobby_id
//             }
//         };
//         sendJsonMessage(request);
//     }, [lobby, sendJsonMessage]);
//
//     useEffect(() => {
//         if (lastJsonMessage != null) {
//             const p = lastJsonMessage as qpRequestTypes;
//             switch (p.msg_type) {
//                 case qpMsgType.qpMsgTypeJoinLobbySuccess: {
//                     const req = p as qpMsgJoinedLobbySuccess;
//                     setLobby(req.payload.lobby);
//                     return;
//                 }
//                 case qpMsgType.qpMsgTypeSendMsgRequest: {
//                     const req = p as qpUserMessage;
//                     setMessageHistory((prev) => prev.concat(req.payload));
//                     return;
//                 }
//                 default: {
//                     console.log(lastJsonMessage);
//                 }
//             }
//         }
//     }, [lastJsonMessage]);
//
//     // const connectionStatus = {
//     //     [ReadyState.CONNECTING]: 'Connecting',
//     //     [ReadyState.OPEN]: 'Open',
//     //     [ReadyState.CLOSING]: 'Closing',
//     //     [ReadyState.CLOSED]: 'Closed',
//     //     [ReadyState.UNINSTANTIATED]: 'Uninstantiated'
//     // }[readyState];
//
//     useEffect(() => {
//         apiServerQuery({
//             gameTypes: []
//         }).then((resp) => {
//             if (!resp.status) {
//                 return;
//             }
//             setAllServers(resp.result ?? []);
//         });
//     }, []);
//
//     const msgs = useMemo(() => {
//         const m = messageHistory;
//         m.reverse();
//         return m;
//     }, [messageHistory]);
//
//     const filteredServers = useMemo(() => {
//         let servers = allServers;
//         if (notFull) {
//             servers = servers.filter((s) => s.players < s.max_players);
//         }
//         if (minPlayers > 0) {
//             servers = servers.filter((s) => s.players >= minPlayers);
//         }
//         if (maxPlayers > 0) {
//             servers = servers.filter((s) => s.players <= maxPlayers);
//         }
//         if (enableGameTypes) {
//             servers = filterServerGameTypes(enableGameTypes, servers);
//         }
//
//         return servers;
//     }, [allServers, notFull, minPlayers, maxPlayers, enableGameTypes]);
//
//     const onGameModesChange = useCallback(
//         (values: qpGameType[]) => {
//             setEnableGameTypes(values);
//         },
//         [setEnableGameTypes]
//     );
//
//     const dataTable = useMemo(() => {
//         return (
//             <DataTable
//                 columns={[
//                     {
//                         label: 'Name',
//                         sortKey: 'name',
//                         tooltip: 'Server Name',
//                         sortable: true,
//                         align: 'left',
//                         queryValue: (row) => row.name
//                     },
//                     {
//                         label: 'Distance',
//                         sortKey: 'distance',
//                         tooltip: 'Distance',
//                         align: 'left',
//                         sortable: true
//                     },
//                     {
//                         label: 'Address',
//                         sortKey: 'addr',
//                         tooltip: 'Address',
//                         align: 'left',
//                         sortable: true
//                     },
//                     {
//                         label: 'Map',
//                         sortKey: 'map',
//                         tooltip: 'Map',
//                         sortable: true,
//                         queryValue: (row) => row.map
//                     },
//                     {
//                         label: 'Players',
//                         sortKey: 'players',
//                         tooltip: 'Players',
//                         sortable: true,
//                         queryValue: (row) => `${row.players}`,
//                         renderer: (row) => {
//                             return `${row.players}/${row.max_players}`;
//                         }
//                     },
//                     {
//                         label: 'Actions',
//                         tooltip: 'Actions',
//                         virtual: true,
//                         virtualKey: 'act',
//                         renderer: (row) => {
//                             return (
//                                 <Button
//                                     variant={'contained'}
//                                     component={Link}
//                                     href={`steam://connect/${row.addr}`}
//                                     endIcon={<ChevronRightIcon />}
//                                 >
//                                     Connect
//                                 </Button>
//                             );
//                         }
//                     }
//                 ]}
//                 defaultSortOrder={'asc'}
//                 defaultSortColumn={'distance'}
//                 rowsPerPage={RowsPerPage.TwentyFive}
//                 rows={filteredServers}
//             />
//         );
//     }, [filteredServers]);
//
//     return (
//         <>
//             <JoinLobbyModal
//                 open={joinModalOpen}
//                 setOpen={setJoinModalOpen}
//                 onSuccess={sendJoinLobbyRequest}
//             />
//             <Grid container paddingTop={3} spacing={2}>
//                 <Grid item xs={12}>
//                     <Grid container spacing={2}>
//                         <Grid item xs={8}>
//                             <Paper elevation={1}>
//                                 <Stack spacing={1}>
//                                     <Heading>{`Lobby Chat`}</Heading>
//
//                                     <Stack
//                                         direction={'column-reverse'}
//                                         sx={{
//                                             height: 200,
//                                             overflow: 'scroll'
//                                         }}
//                                     >
//                                         {msgs.map((msg, i) => {
//                                             return (
//                                                 <Typography
//                                                     key={`msg-${i}`}
//                                                     variant={'body2'}
//                                                 >
//                                                     {msg.created_at} --{' '}
//                                                     {msg.steam_id ??
//                                                         '__lobby__'}{' '}
//                                                     --
//                                                     {msg.message}
//                                                 </Typography>
//                                             );
//                                         })}
//                                     </Stack>
//                                     <Stack direction={'row'}>
//                                         <TextField
//                                             fullWidth
//                                             value={chatInput}
//                                             onChange={(evt) => {
//                                                 setChatInput(evt.target.value);
//                                             }}
//                                             onKeyDown={(evt) => {
//                                                 if (evt.key == 'Enter') {
//                                                     sendMessage();
//                                                 }
//                                             }}
//                                             disabled={!isReady}
//                                         />
//                                         <Button
//                                             color={'success'}
//                                             variant={'contained'}
//                                             onClick={sendMessage}
//                                             disabled={!isReady}
//                                         >
//                                             Send
//                                         </Button>
//                                     </Stack>
//                                 </Stack>
//                             </Paper>
//                         </Grid>
//                         <Grid item xs={4}>
//                             <Paper elevation={1}>
//                                 <Heading>{`Lobby Members (${lobby.clients.length}/${maxLobbyMembers}) (id: ${lobby.lobby_id})`}</Heading>
//                                 <ButtonGroup>
//                                     <Tooltip title={'Join lobby'}>
//                                         <span>
//                                             <IconButton
//                                                 color={'success'}
//                                                 onClick={() => {
//                                                     setJoinModalOpen(true);
//                                                 }}
//                                                 disabled={!isReady}
//                                             >
//                                                 <GroupAddIcon />
//                                             </IconButton>
//                                         </span>
//                                     </Tooltip>
//                                     <Tooltip title={'Leave lobby'}>
//                                         <span>
//                                             <IconButton
//                                                 color={'error'}
//                                                 onClick={sendLeaveLobbyRequest}
//                                                 disabled={
//                                                     !isReady ||
//                                                     lobby.clients.length < 2
//                                                 }
//                                             >
//                                                 <GroupRemoveIcon />
//                                             </IconButton>
//                                         </span>
//                                     </Tooltip>
//                                 </ButtonGroup>
//                                 <List
//                                     sx={{
//                                         width: '100%',
//                                         //maxWidth: 360,
//                                         bgcolor: 'background.paper'
//                                     }}
//                                 >
//                                     {lobby.clients.map((client, i) => {
//                                         const user =
//                                             client.user as unknown as UserProfile;
//                                         return (
//                                             <ListItem
//                                                 disablePadding
//                                                 key={`player-list-${i}`}
//                                             >
//                                                 <ListItemButton>
//                                                     <ListItemIcon>
//                                                         <Tooltip
//                                                             title={'Leader'}
//                                                         >
//                                                             <EmojiEventsIcon />
//                                                         </Tooltip>
//                                                     </ListItemIcon>
//                                                 </ListItemButton>
//                                                 <ListItemText
//                                                     sx={{ width: '100%' }}
//                                                     primary={
//                                                         user.name ??
//                                                         user.steam_id.toString()
//                                                     }
//                                                 />
//                                             </ListItem>
//                                         );
//                                     })}
//                                 </List>
//                             </Paper>
//                         </Grid>
//                     </Grid>
//                 </Grid>
//                 <Grid item xs={12}>
//                     <Paper elevation={1}>
//                         <Heading>Quickplay Filters</Heading>
//                         <Stack spacing={1} direction={'row'} padding={2}>
//                             <TextField
//                                 id="outlined-basic"
//                                 label="Min Players"
//                                 variant="outlined"
//                                 type={'number'}
//                                 value={minPlayers}
//                                 onChange={(evt) => {
//                                     const value = parseInt(evt.target.value);
//                                     if (value && value > 31) {
//                                         return;
//                                     }
//                                     if (maxPlayers > 0 && value > maxPlayers) {
//                                         setMaxPlayers(value);
//                                     }
//                                     setMinPlayers(value ?? 0);
//                                 }}
//                             />
//                             <TextField
//                                 id="outlined-basic"
//                                 label="Max Players"
//                                 variant="outlined"
//                                 type={'number'}
//                                 value={maxPlayers}
//                                 onChange={(evt) => {
//                                     let value = parseInt(evt.target.value);
//                                     if (value && value > 32) {
//                                         return;
//                                     }
//                                     if (value < minPlayers) {
//                                         value = minPlayers;
//                                     }
//                                     setMaxPlayers(value ?? 0);
//                                 }}
//                             />
//                             <FormGroup>
//                                 <FormControlLabel
//                                     control={
//                                         <Switch
//                                             defaultChecked
//                                             value={notFull}
//                                             onChange={(_, checked) => {
//                                                 setNotFull(checked);
//                                             }}
//                                         />
//                                     }
//                                     label="Hide Full"
//                                 />
//                             </FormGroup>
//                             <GameModeSelect onChange={onGameModesChange} />
//                             <FormControl>
//                                 <InputLabel id="queue-mode-label">
//                                     Queue Mode
//                                 </InputLabel>
//                                 <Select<qpAutoQueueMode>
//                                     labelId="queue-mode-label"
//                                     id="queue-mode-select"
//                                     value={queueMode}
//                                     label="Auto Queue Mode"
//                                     onChange={(evt) => {
//                                         setQueueMode(
//                                             evt.target.value as qpAutoQueueMode
//                                         );
//                                     }}
//                                 >
//                                     <MenuItem value={'eager'}>Eager</MenuItem>
//                                     <MenuItem value={'full'}>Full</MenuItem>
//                                 </Select>
//                             </FormControl>
//                             <FormGroup>
//                                 <FormControlLabel
//                                     control={
//                                         <Switch
//                                             value={enableAutoQueue}
//                                             onChange={(_, value) => {
//                                                 setEnableAutoQueue(value);
//                                             }}
//                                         />
//                                     }
//                                     label="Enable Auto Queue"
//                                 />
//                             </FormGroup>
//                         </Stack>
//                     </Paper>
//                 </Grid>
//                 <Grid item xs={12}>
//                     <Paper elevation={1}>
//                         <Heading>Community Servers</Heading>
//                         {dataTable}
//                     </Paper>
//                 </Grid>
//             </Grid>
//         </>
//     );
// };
