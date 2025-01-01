import { transformCreatedAtDate } from '../util/time.ts';
import { apiCall } from './common.ts';

export interface GithubRelease {
    url: string;
    html_url: string;
    assets_url: string;
    upload_url: string;
    tarball_url: string;
    zipball_url: string;
    id: number;
    node_id: string;
    tag_name: string;
    target_commitish: string;
    name: string;
    body: string;
    draft: boolean;
    prerelease: boolean;
    created_at: Date;
    published_at: string;
    author: {
        login: string;
        id: number;
        node_id: string;
        avatar_url: string;
        gravatar_id: string;
        url: string;
        html_url: string;
        followers_url: string;
        following_url: string;
        gists_url: string;
        starred_url: string;
        subscriptions_url: string;
        organizations_url: string;
        repos_url: string;
        events_url: string;
        received_events_url: string;
        type: string;
        site_admin: boolean;
    };
    assets: {
        url: string;
        browser_download_url: string;
        id: number;
        node_id: string;
        name: string;
        label: string;
        state: string;
        content_type: string;
        size: number;
        download_count: number;
        created_at: string;
        updated_at: string;
        uploader: {
            login: string;
            id: number;
            node_id: string;
            avatar_url: string;
            gravatar_id: string;
            url: string;
            html_url: string;
            followers_url: string;
            following_url: string;
            gists_url: string;
            starred_url: string;
            subscriptions_url: string;
            organizations_url: string;
            repos_url: string;
            events_url: string;
            received_events_url: string;
            type: string;
            site_admin: boolean;
        };
    }[];
}

export const getChangelogs = async () => (await apiCall<GithubRelease[]>('/api/changelog')).map(transformCreatedAtDate);

export interface appInfoDetail {
    site_name: string;
    app_version: string;
    link_id: string;
    sentry_dns_web: string;
    asset_url: string;
    patreon_client_id: string;
    patreon_enabled: boolean;
    discord_client_id: string;
    discord_enabled: boolean;
    default_route: string;
    news_enabled: boolean;
    forums_enabled: boolean;
    contests_enabled: boolean;
    wiki_enabled: boolean;
    stats_enabled: boolean;
    servers_enabled: boolean;
    reports_enabled: boolean;
    chatlogs_enabled: boolean;
    demos_enabled: boolean;
    speedruns_enabled: boolean;
}

export const getAppInfo = async () => apiCall<appInfoDetail>('/api/info');
