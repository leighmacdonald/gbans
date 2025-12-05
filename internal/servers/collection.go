package servers

import (
	"context"
	"errors"
	"log/slog"
	"net"
	"slices"
	"strings"

	"github.com/leighmacdonald/steamid/v4/steamid"
	"github.com/ryanuber/go-glob"
	"golang.org/x/sync/errgroup"
)

type broadcastResult struct {
	serverID int
	resp     string
}

type FindOpts struct {
	Name    string
	SteamID steamid.SteamID
	Addr    net.IP
	CIDR    *net.IPNet
}

type FindResult struct {
	Server *Server
	Player *Player
}

type Collection []*Server

func (c Collection) filter(serverIDs []int) Collection {
	var valid Collection
	for _, server := range c {
		if slices.Contains(serverIDs, server.ServerID) {
			valid = append(valid, server)
		}
	}

	return valid
}

// broadcast sends out rcon commands to all provided servers. If no servers are provided it will default to broadcasting
// to every server.
func (c Collection) broadcast(ctx context.Context, cmd string) map[int]string {
	var (
		results         = map[int]string{}
		errGroup, egCtx = errgroup.WithContext(ctx)
		resultChan      = make(chan broadcastResult)
	)

	for _, server := range c {
		errGroup.Go(func() error {
			resp, errExec := server.Exec(egCtx, cmd)
			if errExec != nil {
				if errors.Is(errExec, context.Canceled) {
					return nil
				}

				slog.Error("Failed to exec server command", slog.String("name", server.Name),
					slog.Int("server_id", server.ServerID), slog.String("error", errExec.Error()))

				// Don't error out since we don't want a single servers potentially temporary issue to prevent the rest
				// from executing.
				return nil
			}

			resultChan <- broadcastResult{
				serverID: server.ServerID,
				resp:     resp,
			}

			return nil
		})
	}

	go func() {
		err := errGroup.Wait()
		if err != nil {
			slog.Error("Failed to broadcast command", slog.String("error", err.Error()))
		}

		close(resultChan)
	}()

	for result := range resultChan {
		results[result.serverID] = result.resp
	}

	return results
}

// Find searches the current server state for players matching at least one of the provided criteria.
func (c Collection) find(opts FindOpts) []FindResult {
	var found []FindResult

	for _, server := range c {
		server.RLock()
		for _, player := range server.state.Players {
			matched := false
			if opts.SteamID.Valid() && player.SID == opts.SteamID {
				matched = true
			}

			if opts.Name != "" {
				queryName := opts.Name
				if !strings.HasPrefix(queryName, "*") {
					queryName = "*" + queryName
				}

				if !strings.HasSuffix(queryName, "*") {
					queryName += "*"
				}

				m := glob.Glob(strings.ToLower(queryName), strings.ToLower(player.Name))
				if m {
					matched = true
				}
			}

			if opts.Addr != nil && opts.Addr.Equal(player.IP) {
				matched = true
			}

			if opts.CIDR != nil && opts.CIDR.Contains(player.IP) {
				matched = true
			}

			if matched {
				found = append(found, FindResult{Player: player, Server: server})
			}
		}
		server.RUnlock()
	}

	return found
}

func (c Collection) sortRegion() map[string][]*Server {
	serverMap := map[string][]*Server{}
	for _, server := range c {
		_, exists := serverMap[server.Region]
		if !exists {
			serverMap[server.Region] = []*Server{}
		}

		serverMap[server.Region] = append(serverMap[server.Region], server)
	}

	return serverMap
}

func (c Collection) byServerID(serverID int) (*Server, bool) {
	for _, server := range c {
		if server.ServerID == serverID {
			return server, true
		}
	}

	return nil, false
}

func (c Collection) byName(name string, wildcardOk bool) []*Server {
	var servers []*Server

	if name == "*" && wildcardOk {
		return c
	}
	if !strings.HasPrefix(name, "*") {
		name = "*" + name
	}

	if !strings.HasSuffix(name, "*") {
		name += "*"
	}

	for _, server := range c {
		if glob.Glob(strings.ToLower(name), strings.ToLower(server.ShortName)) ||
			strings.EqualFold(server.ShortName, name) {
			servers = append(servers, server)

			break
		}
	}

	return servers
}

func (c Collection) findExec(ctx context.Context, opts FindOpts, onFoundCmd func(ps FindResult) string) error {
	found := c.find(opts)
	if len(found) == 0 {
		return ErrPlayerNotFound
	}

	var err error
	for _, psi := range found {
		if errRcon := psi.Server.ExecDiscard(ctx, onFoundCmd(psi)); errRcon != nil {
			err = errors.Join(errRcon)
		}
	}

	return err
}
