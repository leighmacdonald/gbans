const baseUrl = () => {
    let returnUrl = window.location.hostname;
    if (window.location.port !== '') {
        returnUrl = `${returnUrl}:${window.location.port}`;
    }
    return `${window.location.protocol}//${returnUrl}`;
};

export const discordLoginURL = (discord_client_id: string) => {
    return (
        'https://discord.com/oauth2/authorize' +
        '?client_id=' +
        discord_client_id +
        '&redirect_uri=' +
        encodeURIComponent(baseUrl() + '/login/discord') +
        '&response_type=code' +
        '&scope=identify'
    );
};
