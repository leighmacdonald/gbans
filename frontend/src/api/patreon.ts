import { apiCall, CallbackLink } from './common.ts';

export interface Relationships {}

export interface PatreonCampaign {
    type: string;
    id: string;
    attributes: {
        created_at: string;
        creation_name: string;
        discord_server_id: string;
        google_analytics_id: string;
        has_rss: boolean;
        has_sent_rss_notify: boolean;
        image_small_url: string;
        image_url: string;
        is_charged_immediately: boolean;
        is_monthly: boolean;
        is_nsfw: boolean;
        main_video_embed: string;
        main_video_url: string;
        one_liner: string;
        patron_count: number;
        pay_per_name: string;
        pledge_url: string;
        published_at: string;
        rss_artwork_url: boolean;
        rss_feed_title: string;
        show_earnings: boolean;
        summary: string;
        thanks_embed: string;
        thanks_msg: string;
        thanks_video_url: string;
        url: string;
        vanity: string;
    };
    // relationships: {
    //     categories?: CategoriesRelationship;
    //     creator?: CreatorRelationship;
    //     rewards?: RewardsRelationship;
    //     goals?: GoalsRelationship;
    //     pledges?: PledgesRelationship;
    //     post_aggregation?: PostAggregationRelationship;
    // };
}

export const apiGetPatreonCampaigns = async () => {
    return apiCall<PatreonCampaign>('/api/patreon/campaigns');
};

export const apiGetPatreonLogin = async () => {
    return apiCall<CallbackLink>('/api/patreon/login');
};

export const apiGetPatreonLogout = async () => {
    return apiCall('/api/patreon/logout');
};
