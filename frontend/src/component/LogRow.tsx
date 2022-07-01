import {
    eventName,
    EventTypeByName,
    Person,
    Server,
    ServerEvent
} from '../api';
import React, { useMemo } from 'react';
import Stack from '@mui/material/Stack';
import Button from '@mui/material/Button';
import Typography from '@mui/material/Typography';
import format from 'date-fns/format';
import parseISO from 'date-fns/parseISO';
import Chip from '@mui/material/Chip';
import { PlayerClassImg } from './PlayerClassImg';
import { ProfileButton } from './ProfileButton';
import { useServerLogQueryCtx } from '../contexts/LogQueryCtx';
import Box from '@mui/material/Box';
import { noop } from 'lodash-es';

export interface EventViewProps {
    event: ServerEvent;
}

export const EventView = ({ event }: EventViewProps) => {
    const e: JSX.Element[] = [];
    if (event?.target?.steam_id && event.target.personaname != '') {
        e.push(
            <ProfileButton
                hideLabel
                source={event.target}
                team={event.team}
                setFilter={noop}
            />
        );
    }
    switch (event.event_type) {
        case EventTypeByName.damage:
            e.push(
                <Typography variant={'subtitle1'} color={'red'}>
                    -{event.damage}
                </Typography>
            );
            break;
        case EventTypeByName.healed:
            e.push(
                <Typography variant={'subtitle1'} color={'green'}>
                    +{event.healing}
                </Typography>
            );
            break;
        case EventTypeByName.change_class:
            e.push(<PlayerClassImg cls={event.player_class} />);
            break;
        case EventTypeByName.spawned_as:
            e.push(<PlayerClassImg cls={event.player_class} />);
            break;
        case EventTypeByName.say:
            e.push(<Typography>{event.meta_data['msg'] as string}</Typography>);
            break;
        case EventTypeByName.say_team:
            e.push(<Typography>{event.meta_data['msg'] as string}</Typography>);
            break;
    }

    if (event?.meta_data) {
        Object.entries(event?.meta_data).forEach((k) => {
            const v = Object.call(event?.meta_data, k) as string[];
            e.push(
                <Chip
                    key={v[0]}
                    //avatar={<Avatar alt="Natacha" src="/static/images/avatar/1.jpg" />}
                    label={`${v[0]}: ${v[1]}`}
                    variant={'filled'}
                />
            );
        });
    }

    return (
        <Stack direction={'row'} spacing={1}>
            {e}
        </Stack>
    );
};

export interface EventTypeLabelProps {
    readonly event_type: number;
}

export const EventTypeLabel = ({ event_type }: EventTypeLabelProps) => {
    return (
        <Typography
            variant={'body1'}
            alignContent={'center'}
            align={'center'}
            sx={{ width: '100px' }}
        >
            {eventName(event_type)}
        </Typography>
    );
};

export interface SteamIDLabelProps {
    source: Person;
}

export const SteamIDLabel = ({ source }: SteamIDLabelProps) => {
    return (
        <Typography
            variant={'body2'}
            alignContent={'center'}
            sx={{ width: '150px', overflow: 'hidden' }}
        >
            {source.steam_id.toString()}
        </Typography>
    );
};

export interface ServerLabelProps {
    server: Server;
}

export const ServerLabel = ({ server }: ServerLabelProps) => {
    const { setSelectedServerIDs } = useServerLogQueryCtx();
    return (
        <Button
            onClick={() => {
                setSelectedServerIDs([server.server_id]);
            }}
        >
            {server.server_name}
        </Button>
    );
};

export interface DateLabelProps {
    date: string;
}

export const DateLabel = ({ date }: DateLabelProps) => {
    return (
        <Typography variant={'subtitle1'} sx={{ width: 100 }}>
            {format(parseISO(date), 'dd/MM/yy hh:mm')}
        </Typography>
    );
};

export interface LogRowProps {
    event: ServerEvent;
}

export const LogRow = ({ event }: LogRowProps): JSX.Element => {
    const { setSteamID } = useServerLogQueryCtx();
    return (
        <Stack
            direction={'row'}
            spacing={2}
            justifyContent="left"
            alignItems="center"
        >
            <ServerLabel server={event.server} />
            <DateLabel date={event.created_on} />
            <EventTypeLabel event_type={event.event_type} />
            {(event.source?.steam_id && event.source?.personaname != '' && (
                <Box sx={{ minWidth: 200 }}>
                    {/*<SteamIDLabel source={event.source} />*/}
                    <ProfileButton
                        source={event.source}
                        team={event.team}
                        setFilter={() => {
                            if (event.source?.steam_id) {
                                setSteamID(event.source?.steam_id);
                            }
                        }}
                    />
                </Box>
            )) || <Box sx={{ minWidth: 200 }} />}
            {<EventView event={event} />}
        </Stack>
    );
};

export interface LogRowsProps {
    events: ServerEvent[];
}

export const LogRows = ({ events }: LogRowsProps) => {
    return useMemo(() => {
        return (
            <Stack>
                {events.map((e) => (
                    <LogRow event={e} key={e.log_id} />
                ))}
            </Stack>
        );
    }, [events]);
};
