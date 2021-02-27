import IProps from "../util/IProps";
import * as React from "react";
import Paginator from "./Paginator";
import {fmtUnix} from "../util/text";
import SearchBar from "./SearchBar";


interface IState {
    current_page: number
    offset: number
    limit: number
    desc: boolean
    q: string
    order_by: string
    bans: IBanState[]
    total: number
}

interface IBanState {
    ban_id: bigint
    steam_id: bigint
    author_id: number
    ban_type: number
    reason: number
    reason_text: string
    note: string
    source: number
    until: number
    created_on: number
    updated_on: number
    personaname: string
    avatar: string
    avatarmedium: string
}

export default class BanBrowser extends React.Component<IProps, IState> {
    constructor(props: IProps) {
        super(props);
        this.onLoad = this.onLoad.bind(this)
        this.onError = this.onError.bind(this)
        this.update = this.update.bind(this)
        this.onSearchInput = this.onSearchInput.bind(this);
        this.state = {
            current_page: 1,
            limit: 10,
            desc: true,
            q: "",
            order_by: "created_on",
            offset: 0,
            bans: [],
            total: 0
        }
    }

    componentDidMount() {
        this.update()
    }

    update() {
        // const args = {
        //     limit: this.state.limit,
        //     offset: this.state.offset,
        //     order_by: this.state.order_by,
        //     desc: this.state.desc
        // };
        //http(vars(Route.APIBans, args), "GET", "", this.onLoad, this.onError)
    }

    changePage(page: number) {
        this.setState({...this.state, offset: this.state.limit * page, current_page: page})
        this.update()
    }

    onSearchInput(evt: any) {
        console.log(evt)
        this.setState({...this.state, q: evt})
        return
    }

    onLoad(json: any) {
        console.log(json)
        const {bans, total} = json;
        this.setState({...this.state, bans: bans, total: total})
    }

    onError(err: string) {
        console.log(err)
    }

    render_rows() {
        const {bans} = this.state;
        return (
            <div className={"grid-y grid-y-padding"} id={"ban_list"}>
                <div className={"cell"}>
                    {bans.map(ban => (
                        <div className={"grid-x ban_row"} key={ban.ban_id.toString()}>
                            <div className={"cell large-5"}>
                                <a href={"https://steamcommunity.com/profiles/" + ban.steam_id}>
                                    <img src={ban.avatar}/>
                                    <span>{ban.personaname}</span>
                                </a>
                            </div>
                            <div className={"cell large-3 text-center"}>
                                {ban.reason_text}
                            </div>
                            <div className={"cell large-2 text-right"}>
                                {fmtUnix(ban.created_on)}
                            </div>
                            <div className={"cell large-2 text-right"}>
                                {ban.until > 0 ? "Permanent" : fmtUnix(ban.until)}
                            </div>
                        </div>
                    ))}
                </div>
            </div>
        )
    }

    render() {
        const {total, current_page} = this.state;
        let t = Math.ceil(total / 10)
        return (
            <div className={"grid-y grid-padding-y"}>
                <div className={"cell"}>
                    <SearchBar onInputChange={this.onSearchInput.bind(this)} />
                </div>
                <div className={"cell"}>
                    {this.render_rows()}
                </div>
                <div className={"cell"}>
                    <Paginator current_page={current_page} total={t} onChange={this.changePage.bind(this)}/>
                </div>
            </div>
        )
    }
}