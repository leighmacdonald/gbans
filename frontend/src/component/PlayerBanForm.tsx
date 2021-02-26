import * as React from "react";
import {SyntheticEvent} from "react";
import IPCIDR from "ip-cidr";
import {format, formatDistanceToNow, fromUnixTime} from "date-fns";
import {chunk} from 'lodash-es';

interface PanFormProps {

}

function ip2int(ip: string): number {
    return ip.split('.').reduce(function (ipInt, octet) {
        return (ipInt << 8) + parseInt(octet, 10)
    }, 0) >>> 0;
}

interface PlayerProfile {
    player: PlayerSummary
    friends: PlayerSummary[]
}

interface PlayerSummary {
    steamid?: string
    communityvisibilitystate?: communityVisibilityState
    profilestate?: profileState
    personaname: string
    realname?: string
    primaryclanid?: string
    timecreated?: number
    avatar?: string
    avatarfull: string
    avatarhash?: string
    profileurl?: string
    loccountrycode?: string
    locstatecode?: string
    loccityid?: number
    personastateflags?: number
}

type BanType = "network" | "steam"

enum profileState {
    Incomplete = 0,
    Setup = 1
}

enum communityVisibilityState {
    Private = 1,
    FriendOnly = 2,
    Public = 3
}

export const PlayerBanForm: React.FC<PanFormProps> = () => {
    const [friendsPage, setFriendsPage] = React.useState<number>(0);
    const [showFriends, setShowFriends] = React.useState<boolean>(false);
    const [fSteam, setFSteam] = React.useState<string>("https://steamcommunity.com/id/SQUIRRELLY/");
    const [duration, setDuration] = React.useState<string>("48h")
    const [reasonText, setReasonText] = React.useState<string>("")
    const [network, setNetwork] = React.useState<string>("")
    const [networkSize, setNetworkSize] = React.useState<number>(0)
    const [banType, setBanType] = React.useState<BanType>("steam")
    const [profile, setProfile] = React.useState<PlayerProfile>({
        player: {
            personaname: "Player Name",
            avatarfull: "https://cdn.cloudflare.steamstatic.com/steamcommunity/public" +
                "/images/avatars/30/3077b41fd6ae20862c69fddfef1ff514ecb375cb_full.jpg"
        }, friends: [],
    });
    const loadPlayerSummary = async () => {
        // TODO filter to known formats only
        const resp = await fetch(`/api/v1/profile?query=${fSteam}`, {
            mode: "cors",
            headers: {
                "Content-Type": "application/json"
            }
        });
        if (!resp.ok) {
            console.log("Failed to lookup user profile");
            return;
        }
        const summary = await resp.json();
        setProfile(summary)
    };
    const handleUpdateFSteam = React.useCallback(loadPlayerSummary, [fSteam]);
    const handleUpdateNetwork = (evt: SyntheticEvent) => {
        const value = (evt.target as HTMLInputElement).value;
        setNetwork(value);
        if (value !== "") {
            try {
                const cidr = new IPCIDR(value)
                if (cidr != undefined) {
                    setNetworkSize((ip2int(cidr?.end()) - ip2int(cidr?.start())) + 1)
                }
            } catch (e) {
                // TypeError on invalid input we can ignore
            }
        }
    }

    const handleUpdateReasonText = (evt: SyntheticEvent) => {
        const value = (evt.target as HTMLInputElement).value;
        setReasonText(value);
    }

    const handleUpdateDuration = (evt: SyntheticEvent) => {
        const value = (evt.target as HTMLInputElement).value;
        setDuration(value);
    }

    const onChangeFStream = (evt: SyntheticEvent) => {
        setFSteam((evt.target as HTMLInputElement).value)
    }

    return (<>
            <div className={"grid-y grid-padding-y"}>
                <div className={"grid-x grid-padding-x"}>
                    <div className={"cell medium-6"}>
                        <div className={"grid-y"}>
                            <h2>Ban Details</h2>
                        </div>
                        <div className={"grid-y"}>
                            <div className={"cell"}>
                                <label form={"fSteam"}>Steam ID / Profile URL</label>
                            </div>
                            <div className={"cell"}>
                                <input name={"fSteam"} type={"text"} value={fSteam} onChange={onChangeFStream}
                                       onBlur={handleUpdateFSteam}/>
                            </div>
                            <div className={"cell"}>
                                <div className={"grid-x"}>
                                    <fieldset className="cell">
                                        <legend>Ban Mode</legend>
                                        <p>SteamID is optional for network bans, however it can be used to trace the
                                            initial culprit of a ban.
                                        </p>
                                        <input type={"radio"} name={"ban_type"} value={"steam"}
                                               checked={banType == "steam"} id={"steam"} onChange={(() => {
                                            setBanType("steam")
                                        })}/><label htmlFor={"steam"}>Steam Ban</label>
                                        <input type={"radio"} name={"ban_type"} value={"network"}
                                               checked={banType == "network"} id={"network"} onChange={() => {
                                            setBanType("network")
                                        }}/><label htmlFor={"network"}>Network Ban</label>
                                    </fieldset>
                                </div>
                            </div>
                            {banType == "network" && <>
                                <div className={"cell"}>
                                    <label form={"fSteam"}>Network Range (CIDR Format)</label>
                                </div>
                                <div className={"cell"}>
                                    <input name={"network"} type={"text"} value={network} placeholder={"12.34.56.78/32"}
                                           onChange={handleUpdateNetwork}
                                           title={"Must be CIDR format with 2 digit mask"}
                                           pattern={"^(\\d{1,3}[\\.\\/]){4}\\d{2}$"}
                                    />
                                    <p>Current number of hosts in range: {networkSize}</p>
                                </div>
                            </>
                            }
                            <div className={"cell"}>
                                <label form={"fSteam"}>Reason</label>
                            </div>
                            <div className={"cell"}>
                                <input name={"reason_text"} type={"text"} value={reasonText}
                                       onChange={handleUpdateReasonText}/>
                            </div>

                            <div className={"cell"}>
                                <label form={"duration"}>Duration</label>
                            </div>
                            <div className={"cell"}>
                                <select onChange={handleUpdateDuration} value={duration}>
                                    {["15m", "6h", "12h", "24h", "48h", "72h", "1w", "2w", "1m", "6m", "1y", "∞", "custom"].map((v) => {
                                            return <option key={`time-${v}`} value={v}>{v}</option>
                                        }
                                    )}
                                </select>
                                {duration === "custom" && (
                                    <label form={"duration"}>
                                        Custom Duration
                                        <input name={"duration"} type={"text"} placeholder={"5d"}/>
                                    </label>
                                )}
                            </div>
                            <div className={"cell"}>
                                <a className={"button"}>Submit Ban <i className={"fi-flag"}
                                                                      style={{"color": "red"}}/></a>
                            </div>
                        </div>
                    </div>
                    <div className={"cell medium-6"}>
                        {profile?.player && profile?.player?.avatarfull &&
                        <div className={"grid-y"}>
                            <div className={"cell"}>
                                <div className="expanded button-group">
                                    <a className={!friendsPage ? "button": "button secondary"} onClick={() => {
                                        setShowFriends(false);
                                    }}>Profile</a>
                                    <a className={friendsPage ? "button": "button secondary"} onClick={() => {
                                        setShowFriends(true);
                                    }}>Friends ({profile?.friends?.length ?? "n/a"})</a>
                                </div>
                            </div>
                            {!showFriends && <>
                                <div className={"cell"}>
                                    <figure className={"text-center"}>
                                        <img src={profile.player.avatarfull} alt={"Current user avatar"}/>
                                        <figcaption>{profile.player.steamid}</figcaption>
                                    </figure>
                                </div>
                                <div className={"cell"}>
                                    <h4 className={"text-center"}>{profile.player.personaname}</h4>
                                    {profile.player.realname != "" && (
                                        <h4 className={"text-center"}><small>{profile.player.realname}</small></h4>
                                    )}
                                </div>
                                <div className={"cell"}>
                                    <dl>
                                        <dt>Community Visibility State</dt>
                                        <dd>{profile.player.communityvisibilitystate == communityVisibilityState.Public ? "Public" : "Private"}</dd>

                                        <dt>Profile State</dt>
                                        <dd>{profile.player.profilestate ? "Configured" : "Non-Configured"}</dd>
                                        {profile.player?.timecreated && <>
                                            <dt>Created</dt>
                                            <dd>{format(fromUnixTime(profile.player.timecreated), "dd-MM-Y")}</dd>
                                            <dt>Age</dt>
                                            <dd>{formatDistanceToNow(fromUnixTime(profile.player.timecreated), {
                                                addSuffix: false,
                                                includeSeconds: true
                                            })}</dd>
                                        </>}
                                        {profile.player.loccountrycode != "" && <>
                                            <dt>Country</dt>
                                            <dd>{profile.player.loccountrycode}</dd>
                                        </>}
                                        {profile.player.locstatecode != "" && <>
                                            <dt>State/Province</dt>
                                            <dd>{profile.player.locstatecode}</dd>
                                        </>}
                                    </dl>
                                </div>
                            </>}
                            {showFriends &&
                            <>
                                <div className="expanded button-group">
                                    {chunk(profile.friends, 25).map((_, index) => {
                                        return (
                                            <a key={`button-${index}`}
                                               className={index == friendsPage ? "button" : "button secondary"}
                                               onClick={(() => {
                                                   setFriendsPage(index)
                                               })}>{index}</a>
                                        )
                                    })}
                                </div>
                                <div className="grid-y grid-padding-y">
                                    {profile.friends.filter((_, index) => {
                                        return index + 1 >= Math.max(friendsPage, 0) * 25 && index + 1 <= Math.max(friendsPage + 1, 0) * 25
                                    }).map((value) => {
                                        return <div className={"grid-x grid-padding-x"} key={`friend-${value.steamid}`}>
                                            <a className={"cell"} target={"_blank"} style={{"display": "inline-block"}}
                                               href={value.profileurl}>
                                                <img src={value.avatar} alt={"Profile Avatar"}/> {value.personaname}
                                            </a>
                                        </div>
                                    })}
                                </div>
                            </>}
                        </div>
                        }
                    </div>
                </div>
            </div>
        </>
    )
}