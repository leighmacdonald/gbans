import { useEffect, useRef } from 'react';
import { useLocation } from 'react-router';
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

export const useScrollToLocation = () => {
    const scrolledRef = useRef(false);
    const { hash } = useLocation();
    const hashRef = useRef(hash);

    useEffect(() => {
        if (hash) {
            // We want to reset if the hash has changed
            if (hashRef.current !== hash) {
                hashRef.current = hash;
                scrolledRef.current = false;
            }

            // only attempt to scroll if we haven't yet (this could have just reset above if hash changed)
            if (!scrolledRef.current) {
                const id = hash.replace('#', '');
                const element = document.getElementById(id);
                if (element) {
                    element.scrollIntoView({ behavior: 'smooth' });
                    scrolledRef.current = true;
                }
            }
        }
    });
};

export const createExternalLinks = (steam_id_str: string): LinkProps[] => {
    const steam_id = new SteamID(steam_id_str);

    return [
        {
            title: 'Steam',
            url: `https://steamcommunity.com/profiles/${steam_id.getSteamID64()}`
        },
        {
            title: 'RGL',
            url: `https://rgl.gg/Public/PlayerProfile.aspx?p=${steam_id.getSteamID64()}`
        },
        {
            title: 'UGC',
            url: `https://www.ugcleague.com/players_page.cfm?player_id=${steam_id.getSteamID64()}`
        },
        {
            title: 'OzFortress',
            url: `https://ozfortress.com/users?q=${steam_id.getSteamID64()}`
        },
        {
            title: 'logs.tf',
            url: `https://logs.tf/profile/${steam_id.getSteamID64()}`
        },
        {
            title: 'demos.tf',
            url: `https://demos.tf/profiles/${steam_id.getSteamID64()}`
        },
        {
            title: 'backpack.tf',
            url: `https://backpack.tf/profiles/${steam_id.getSteamID64()}`
        },
        {
            title: 'trends.tf',
            url: `https://trends.tf/player/${steam_id.getSteamID64()}`
        },
        {
            title: 'steamid.uk',
            url: `https://steamid.uk/profile/${steam_id.getSteamID64()}`
        },
        {
            title: 'SteamRep',
            url: `https://steamrep.com/profiles/${steam_id.getSteamID64()}`
        }
    ];
};
