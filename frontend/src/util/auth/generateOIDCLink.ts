export const generateOIDCLink = (returnPath: string): string => {
    let returnUrl = window.location.hostname;
    if (window.location.port !== '') {
        returnUrl = `${returnUrl}:${window.location.port}`;
    }
    // Don't redirect loop to /login
    const returnTo = `${window.location.protocol}//${returnUrl}/auth/callback?return_url=${returnPath !== '/login' ? returnPath : '/'}`;

    return [
        'https://steamcommunity.com/openid/login',
        `?openid.ns=${encodeURIComponent('http://specs.openid.net/auth/2.0')}`,
        '&openid.mode=checkid_setup',
        `&openid.return_to=${encodeURIComponent(returnTo)}`,
        `&openid.realm=${encodeURIComponent(`${window.location.protocol}//${window.location.hostname}`)}`,
        `&openid.ns.sreg=${encodeURIComponent('http://openid.net/extensions/sreg/1.1')}`,
        `&openid.claimed_id=${encodeURIComponent('http://specs.openid.net/auth/2.0/identifier_select')}`,
        `&openid.identity=${encodeURIComponent('http://specs.openid.net/auth/2.0/identifier_select')}`
    ].join('');
};
