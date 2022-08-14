import { createBrowserHistory } from 'history';
import SteamID from 'steamid';
export default createBrowserHistory();

export const to = (url: string) => {
    window.open(url, '_self');
};

export interface LinkProps {
    title: string;
    url: string;
}

export const createExternalLinks = (steam_id: SteamID): LinkProps[] => {
    return [
        {
            title: 'Steam',
            url: `https://steamcommunity.com/profiles/${steam_id}`
        },
        {
            title: 'RGL',
            url: `https://rgl.gg/Public/PlayerProfile.aspx?p=${steam_id}`
        },
        {
            title: 'UGC',
            url: `https://www.ugcleague.com/players_page.cfm?player_id=${steam_id}`
        },
        {
            title: 'OzFortress',
            url: `https://ozfortress.com/users?q=${steam_id}`
        },
        { title: 'logs.tf', url: `https://logs.tf/profile/${steam_id}` },
        { title: 'demos.tf', url: `https://demos.tf/profiles/${steam_id}` },
        {
            title: 'backpack.tf',
            url: `https://backpack.tf/profiles/${steam_id}`
        },
        { title: 'trends.tf', url: `https://trends.tf/player/${steam_id}` },
        { title: 'steamid.uk', url: `https://steamid.uk/profile/${steam_id}` },
        { title: 'SteamRep', url: `https://steamrep.com/profiles/${steam_id}` }
    ];
};
