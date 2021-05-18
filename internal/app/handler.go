package app

import (
	"context"
	"github.com/leighmacdonald/gbans/internal/action"
	"github.com/leighmacdonald/gbans/internal/consts"
	"github.com/leighmacdonald/gbans/internal/store"
	"net"
)

func errResult(e error) action.Result {
	return action.Result{Err: e, Message: "", Value: nil}
}
func okResult(v interface{}) action.Result {
	return action.Result{Err: nil, Message: "", Value: v}
}

var invalidArgs = errResult(action.ErrInvalidArgs)
var invalidSteamID = errResult(consts.ErrInvalidSID)

func onActionMute(ctx context.Context, act *action.Action) {
	args, ok := act.Args.(action.MuteRequest)
	if !ok {
		act.SetResult(invalidArgs)
		return
	}
	res, err := mute(ctx, args)
	if err != nil {
		act.SetResult(errResult(err))
		return
	}
	act.SetResult(okResult(res))
}
func onActionKick(ctx context.Context, act *action.Action) {
	args, ok := act.Args.(action.KickRequest)
	if !ok {
		act.SetResult(invalidArgs)
		return
	}
	res, err := kick(ctx, &args)
	if err != nil {
		act.SetResult(errResult(err))
		return
	}
	act.SetResult(okResult(res))
}

func onActionBan(ctx context.Context, act *action.Action) {
	args, ok := act.Args.(action.BanRequest)
	if !ok {
		act.SetResult(invalidArgs)
		return
	}
	res, err := ban(ctx, args)
	if err != nil {
		act.SetResult(errResult(err))
		return
	}
	act.SetResult(okResult(res))
}

func onActionBanNet(ctx context.Context, act *action.Action) {
	args, ok := act.Args.(action.BanNetRequest)
	if !ok {
		act.SetResult(invalidArgs)
		return
	}
	res, err := banNetwork(ctx, &args)
	if err != nil {
		act.SetResult(errResult(err))
		return
	}
	act.SetResult(okResult(res))
}

func onActionFind(ctx context.Context, act *action.Action) {
	args, ok := act.Args.(action.FindRequest)
	if !ok {
		act.SetResult(invalidArgs)
		return
	}
	res := FindPlayer(ctx, args.Query, "")
	act.SetResult(okResult(res))
}

func onActionFindByCIDR(ctx context.Context, act *action.Action) {
	args, ok := act.Args.(action.FindCIDRRequest)
	if !ok {
		act.SetResult(invalidArgs)
		return
	}
	res, err := FindPlayerByCIDR(ctx, args.CIDR)
	if err != nil {
		act.SetResult(errResult(err))
		return
	}
	act.SetResult(okResult(res))
}

func onActionCheckFilter(ctx context.Context, act *action.Action) {
	args, ok := act.Args.(action.FilterCheckRequest)
	if !ok {
		act.SetResult(invalidArgs)
		return
	}
	res, err := filterCheck(ctx, args)
	if err != nil {
		act.SetResult(errResult(err))
		return
	}
	act.SetResult(okResult(res))
}

func onActionAddFilter(ctx context.Context, act *action.Action) {
	args, ok := act.Args.(action.FilterAddRequest)
	if !ok {
		act.SetResult(invalidArgs)
		return
	}
	res, err := filterAdd(ctx, args)
	if err != nil {
		act.SetResult(errResult(err))
		return
	}
	act.SetResult(okResult(res))
}

func onActionDelFilter(ctx context.Context, act *action.Action) {
	args, ok := act.Args.(action.FilterDelRequest)
	if !ok {
		act.SetResult(invalidArgs)
		return
	}
	res, err := filterDel(ctx, args)
	if err != nil {
		act.SetResult(errResult(err))
		return
	}
	act.SetResult(okResult(res))
}

func onActionGetPersonByID(ctx context.Context, act *action.Action) {
	args, ok := act.Args.(action.GetPersonByIDRequest)
	if !ok {
		act.SetResult(invalidArgs)
		return
	}
	sid, err := args.Target.SID64()
	if err != nil || !sid.Valid() {
		act.SetResult(errResult(consts.ErrInvalidSID))
		return
	}
	res, err2 := store.GetPersonBySteamID(ctx, sid)
	if err2 != nil {
		act.SetResult(errResult(err2))
		return
	}
	act.SetResult(okResult(res))
}

func onActionGetOrCreatePersonByID(ctx context.Context, act *action.Action) {
	args, ok := act.Args.(action.GetOrCreatePersonByIDRequest)
	if !ok {
		act.SetResult(invalidArgs)
		return
	}
	sid, err := args.Target.SID64()
	if err != nil || !sid.Valid() {
		act.SetResult(errResult(consts.ErrInvalidSID))
		return
	}
	res, err2 := store.GetOrCreatePersonBySteamID(ctx, sid)
	if err2 != nil {
		act.SetResult(errResult(err2))
		return
	}
	act.SetResult(okResult(res))
}

func onActionUnban(ctx context.Context, act *action.Action) {
	args, ok := act.Args.(action.UnbanRequest)
	if !ok {
		act.SetResult(invalidArgs)
		return
	}
	sid, err := args.Target.SID64()
	if err != nil || !sid.Valid() {
		act.SetResult(errResult(consts.ErrInvalidSID))
		return
	}
	res, err2 := unban(ctx, args)
	if err2 != nil {
		act.SetResult(errResult(err2))
		return
	}
	act.SetResult(okResult(res))
}

func onActionSetSteamID(ctx context.Context, act *action.Action) {
	args, ok := act.Args.(action.SetSteamIDRequest)
	if !ok {
		act.SetResult(invalidArgs)
		return
	}
	sid, err := args.Target.SID64()
	if err != nil || !sid.Valid() {
		act.SetResult(errResult(consts.ErrInvalidSID))
		return
	}
	res, err2 := setSteam(ctx, args)
	if err2 != nil {
		act.SetResult(errResult(err2))
		return
	}
	act.SetResult(okResult(res))
}

func onActionSay(ctx context.Context, act *action.Action) {
	args, ok := act.Args.(action.SayRequest)
	if !ok {
		act.SetResult(invalidArgs)
		return
	}
	res, err2 := say(ctx, args)
	if err2 != nil {
		act.SetResult(errResult(err2))
		return
	}
	act.SetResult(okResult(res))
}

func onActionCSay(ctx context.Context, act *action.Action) {
	args, ok := act.Args.(action.CSayRequest)
	if !ok {
		act.SetResult(invalidArgs)
		return
	}
	res, err2 := csay(ctx, args)
	if err2 != nil {
		act.SetResult(errResult(err2))
		return
	}
	act.SetResult(okResult(res))
}

func onActionPSay(ctx context.Context, act *action.Action) {
	args, ok := act.Args.(action.PSayRequest)
	if !ok {
		act.SetResult(invalidArgs)
		return
	}
	res, err2 := psay(ctx, args)
	if err2 != nil {
		act.SetResult(errResult(err2))
		return
	}
	act.SetResult(okResult(res))
}

func onActionGetBan(ctx context.Context, act *action.Action) {
	args, ok := act.Args.(action.GetBanRequest)
	if !ok {
		act.SetResult(invalidArgs)
		return
	}
	sid, err := args.Target.SID64()
	if err != nil {
		act.SetResult(invalidSteamID)
		return
	}
	res, err2 := store.GetBanBySteamID(ctx, sid, false)
	if err2 != nil {
		act.SetResult(errResult(err2))
		return
	}
	act.SetResult(okResult(res))
}

func onActionGetBanNet(ctx context.Context, act *action.Action) {
	args, ok := act.Args.(action.GetBanNetRequest)
	if !ok {
		act.SetResult(invalidArgs)
		return
	}
	i := net.ParseIP(string(args.Target))
	res, err2 := store.GetBanNet(ctx, i)
	if err2 != nil {
		act.SetResult(errResult(err2))
		return
	}
	act.SetResult(okResult(res))
}

func onActionGetHistoryIP(ctx context.Context, act *action.Action) {
	args, ok := act.Args.(action.GetHistoryIPRequest)
	if !ok {
		act.SetResult(invalidArgs)
		return
	}
	sid, err := args.Target.SID64()
	if err != nil || !sid.Valid() {
		act.SetResult(invalidSteamID)
		return
	}
	res, err2 := store.GetIPHistory(ctx, sid)
	if err2 != nil {
		act.SetResult(errResult(err2))
		return
	}
	act.SetResult(okResult(res))
}

func onActionGetHistoryChat(ctx context.Context, act *action.Action) {
	args, ok := act.Args.(action.GetHistoryChatRequest)
	if !ok {
		act.SetResult(invalidArgs)
		return
	}
	sid, err := args.Target.SID64()
	if err != nil || !sid.Valid() {
		act.SetResult(invalidSteamID)
		return
	}
	res, err2 := store.GetChatHistory(ctx, sid)
	if err2 != nil {
		act.SetResult(errResult(err2))
		return
	}
	act.SetResult(okResult(res))
}

func onActionGetASNRecord(ctx context.Context, act *action.Action) {
	args, ok := act.Args.(action.GetASNRecordRequest)
	if !ok {
		act.SetResult(invalidArgs)
		return
	}
	res, err2 := store.GetASNRecord(ctx, net.ParseIP(args.IPAddr))
	if err2 != nil {
		act.SetResult(errResult(err2))
		return
	}
	act.SetResult(okResult(res))
}

func onActionGetLocationRecord(ctx context.Context, act *action.Action) {
	args, ok := act.Args.(action.GetLocationRecordRequest)
	if !ok {
		act.SetResult(invalidArgs)
		return
	}
	res, err2 := store.GetLocationRecord(ctx, net.ParseIP(args.IPAddr))
	if err2 != nil {
		act.SetResult(errResult(err2))
		return
	}
	act.SetResult(okResult(res))
}

func onActionGetProxyRecord(ctx context.Context, act *action.Action) {
	args, ok := act.Args.(action.GetProxyRecordRequest)
	if !ok {
		act.SetResult(invalidArgs)
		return
	}
	res, err2 := store.GetProxyRecord(ctx, net.ParseIP(args.IPAddr))
	if err2 != nil {
		act.SetResult(errResult(err2))
		return
	}
	act.SetResult(okResult(res))
}

func onActionServers(ctx context.Context, act *action.Action) {
	res, err := store.GetServers(ctx)
	if err != nil {
		act.SetResult(errResult(err))
		return
	}
	act.SetResult(okResult(res))
}

func onActionServerByName(ctx context.Context, act *action.Action) {
	args, ok := act.Args.(action.ServerByNameRequest)
	if !ok {
		act.SetResult(invalidArgs)
		return
	}
	res, err := store.GetServerByName(ctx, args.ServerName)
	if err != nil {
		act.SetResult(errResult(err))
		return
	}
	act.SetResult(okResult(res))
}
