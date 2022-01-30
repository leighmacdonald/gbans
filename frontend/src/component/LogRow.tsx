import { EventType, Person, Server, ServerEvent, Team } from '../api';
import React from 'react';
import Stack from '@mui/material/Stack';
import Button from '@mui/material/Button';
import Paper from '@mui/material/Paper';
import Typography from '@mui/material/Typography';
import format from 'date-fns/format';
import parseISO from 'date-fns/parseISO';
import Chip from '@mui/material/Chip';

export interface EventViewProps {
    event: ServerEvent;
}

export const EventView = ({ event }: EventViewProps) => {
    const e: JSX.Element[] = [];
    switch (event.event_type) {
        case EventType.Say:
            e.push(<Typography>{event.extra ?? ''}</Typography>);
            break;
        case EventType.SayTeam:
            e.push(<Typography>{event.extra ?? ''}</Typography>);
            break;
        default:
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
            break;
    }
    return (
        <Stack direction={'row'} spacing={1}>
            {e}
        </Stack>
    );
};

export interface EventTypeLabelProps {
    event_type: number;
}

export const EventTypeLabel = ({ event_type }: EventTypeLabelProps) => {
    return (
        <Typography
            variant={'body1'}
            alignContent={'center'}
            sx={{ width: '40px' }}
        >
            {event_type}
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
            {source.steam_id}
        </Typography>
    );
};

export interface PersonaNameLabelProps {
    source: Person;
    team: Team;
}

export const teamColour = (team: Team): string => {
    switch (team) {
        case Team.BLU:
            return '#99C2D8';
        case Team.RED:
            return '#FB524F';
        default:
            return '#b98e64';
    }
};

export const PersonaNameLabel = ({ source, team }: PersonaNameLabelProps) => {
    return (
        <Typography
            variant={'body1'}
            alignContent={'center'}
            sx={{ width: '200px', overflow: 'hidden', color: teamColour(team) }}
        >
            {source.personaname}
        </Typography>
    );
};

export interface ServerLabelProps {
    server: Server;
}

export const ServerLabel = ({ server }: ServerLabelProps) => {
    return <Button>{server.server_name}</Button>;
};

export interface DateLabelProps {
    date: string;
}

export const DateLabel = ({ date }: DateLabelProps) => {
    return (
        <Typography variant={'subtitle1'} sx={{ width: '200px' }}>
            {format(parseISO(date), 'dd/MM/yy hh:mm')}
        </Typography>
    );
};

export interface LogRowProps {
    event: ServerEvent;
}

export const LogRow = ({ event }: LogRowProps): JSX.Element => {
    return (
        <Paper>
            <Stack
                direction={'row'}
                spacing={1}
                justifyContent="left"
                alignItems="center"
            >
                <ServerLabel server={event.server} />
                <DateLabel date={event.created_on} />
                <EventTypeLabel event_type={event.event_type} />
                {event.source?.steam_id && (
                    <>
                        <SteamIDLabel source={event.source} />
                        <PersonaNameLabel
                            source={event.source}
                            team={event.team}
                        />
                    </>
                )}
                {<EventView event={event} />}
            </Stack>
        </Paper>
    );
};

export interface LogRowsProps {
    events: ServerEvent[];
}

export const LogRows = ({ events }: LogRowsProps) => {
    return (
        <Stack>
            {events.map((e) => (
                <LogRow event={e} key={e.log_id} />
            ))}
        </Stack>
    );
};
