import * as React from "react";
import Paginator from "./Paginator";
import SearchBar from "./SearchBar";
import {SyntheticEvent, useEffect} from "react";
import {Link} from "react-router-dom"
import {apiGetBans, BannedPerson, IAPIResponseBans} from "../util/api";
import {parseDateTime, renderTimeDistance} from "../util/text";
import {differenceInYears} from "date-fns";

export const BanList = () => {
    const [page, setPage] = React.useState<number>(0)
    // @ts-ignore
    const [limit, setLimit] = React.useState<number>(25)
    // @ts-ignore
    const [sortDesc, setSortDesc] = React.useState<boolean>(true)
    // @ts-ignore
    const [orderBy, setOrderBy] = React.useState<string>("created_on")
    const [bans, setBans] = React.useState<BannedPerson[]>([])
    const [queryStr, setQueryStr] = React.useState<string>("")
    const [total, setTotal] = React.useState<number>(0)

    const changePage = (new_page: number) => {
        setPage(new_page)
    }

    useEffect(() => {
        const getBans = async () => {
            const resp = await apiGetBans({
                sort_desc: sortDesc,
                limit: limit,
                offset: page,
                order_by: orderBy,
                query: queryStr
            }) as IAPIResponseBans
            setBans(resp.bans ?? [])
            setTotal(resp.total ?? 0)
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
                        <div className={"grid-x"}>
                            <div className={"cell large-5"}><span>Profile</span></div>
                            <div className={"cell large-3 text-center"}><span>Reason</span></div>
                            <div className={"cell large-2 text-right"}><span>Created On</span></div>
                            <div className={"cell large-2 text-right"}><span>Valid Until</span></div>
                        </div>
                        {bans.map(ban => (
                            <Link to={`/ban/${ban.ban.ban_id}`} key={ban.ban.ban_id?.toString()}>
                                <div className={"grid-x ban_row"}>
                                    <div className={"cell large-5"}>
                                        <img src={ban.person.avatar} alt={"Player Avatar"}/>
                                        <span>{ban.person.personaname}</span>
                                    </div>
                                    <div className={"cell large-3 text-center"}>
                                        {ban.ban.reason_text}
                                    </div>
                                    <div className={"cell large-2 text-right"}>
                                        {renderTimeDistance(ban.ban.created_on)}
                                    </div>
                                    <div className={"cell large-2 text-right"}>
                                        {differenceInYears(parseDateTime(ban.ban.valid_until), new Date()) > 5
                                            ? "Permanent"
                                            : renderTimeDistance(parseDateTime(ban.ban.valid_until))}
                                    </div>
                                </div>
                            </Link>
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