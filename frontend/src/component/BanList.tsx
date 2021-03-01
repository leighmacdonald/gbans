import * as React from "react";
import Paginator from "./Paginator";
import SearchBar from "./SearchBar";
import {SyntheticEvent, useEffect} from "react";
import {apiCall} from "../util/network";
import {IAPIRequestBans, IAPIResponseBans, IBanState} from "../util/api";

export const BanList = () => {
    const [page, setPage] = React.useState<number>(0)
    // @ts-ignore
    const [limit, setLimit] = React.useState<number>(25)
    // @ts-ignore
    const [sortDesc, setSortDesc] = React.useState<boolean>(true)
    // @ts-ignore
    const [orderBy, setOrderBy] = React.useState<string>("created_on")
    const [bans, setBans] = React.useState<IBanState[]>([])
    const [queryStr, setQueryStr] = React.useState<string>("")
    const [total, setTotal] = React.useState<number>(0)

    const changePage = (new_page: number) => {
        setPage(new_page)
    }

    useEffect(() => {
        const getBans = async () => {
            const resp = await apiCall<IAPIResponseBans, IAPIRequestBans>("/api/v1/bans", "POST", {
                sort_desc: sortDesc,
                limit: limit,
                offset: page,
                order_by: orderBy,
                query: queryStr
            })
            console.log(resp)
            if (!resp.status) {
                // TODO show error
                return
            }
            setBans((resp.json as IAPIResponseBans).bans ?? [])
            setTotal((resp.json as IAPIResponseBans).total ?? 0)
        }
        getBans()
    }, [])

    let t = Math.ceil(total / 10)
    return (
        <div className={"grid-y grid-padding-y"}>
            <div className={"cell"}>
                <SearchBar onInputChange={(e: SyntheticEvent) => {
                    setQueryStr((e.target as HTMLInputElement).value)
                }}/>
            </div>
            <div className={"cell"}>
                <div className={"grid-y grid-y-padding"} id={"ban_list"}>
                    <div className={"cell"}>
                        {bans.map(ban => (
                            <div className={"grid-x ban_row"} key={ban.ban_id.toString()}>
                                <div className={"cell large-5"}>
                                    <a href={"https://steamcommunity.com/profiles/" + ban.steam_id}>
                                        <img src={ban.avatar} alt={"Player Avatar"}/>
                                        <span>{ban.personaname}</span>
                                    </a>
                                </div>
                                <div className={"cell large-3 text-center"}>
                                    {ban.reason_text}
                                </div>
                                <div className={"cell large-2 text-right"}>
                                    {ban.created_on}
                                </div>
                                <div className={"cell large-2 text-right"}>
                                    {ban.valid_until > 0 ? "Permanent" : ban.valid_until}
                                </div>
                            </div>
                        ))}
                    </div>
                </div>
            </div>
            <div className={"cell"}>
                <Paginator current_page={page} total={t} onChange={changePage}/>
            </div>
        </div>
    )
}