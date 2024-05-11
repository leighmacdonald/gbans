package network

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"os"
	"sync"
	"time"

	"github.com/leighmacdonald/gbans/internal/domain"
	"github.com/leighmacdonald/gbans/pkg/log"
	"github.com/viant/afs/scp"
	"github.com/viant/afs/storage"
	"golang.org/x/crypto/ssh"
)

var (
	errInsufficientAuthMethod = errors.New("ssh password or private key missing")
	errReadPrivateKey         = errors.New("failed to read private key contents")
	errParsePrivateKey        = errors.New("failed to parse private key")
	errUsername               = errors.New("invalid username")
	errConnect                = errors.New("failed to connect to ssh server")
)

type OnClientConnect func(ctx context.Context, client storage.Storager, server domain.Server) error

type SCPExecer struct {
	serversUsecase domain.ServersUsecase
	configUsecase  domain.ConfigUsecase
	onConnect      OnClientConnect
}

func NewSCPExecer(configUsecase domain.ConfigUsecase, serversUsecase domain.ServersUsecase, onConnect OnClientConnect) SCPExecer {
	return SCPExecer{
		configUsecase:  configUsecase,
		serversUsecase: serversUsecase,
		onConnect:      onConnect,
	}
}

func (f SCPExecer) Start(ctx context.Context) {
	updateTicker := time.NewTicker(time.Second)

	for {
		select {
		case <-updateTicker.C:
			if errUpdate := f.update(ctx); errUpdate != nil {
				slog.Error("Error querying ssh demo", log.ErrAttr(errUpdate))

				continue
			}
		case <-ctx.Done():
			return
		}
	}
}

func (f SCPExecer) update(ctx context.Context) error {
	servers, _, errServers := f.serversUsecase.GetServers(ctx, domain.ServerQueryFilter{})
	if errServers != nil {
		return errServers
	}

	sshConfig := f.configUsecase.Config().SSH

	wg := sync.WaitGroup{}
	for _, server := range servers {
		wg.Add(1)

		go func(localServer domain.Server) {
			defer wg.Done()

			scpClient, errClient := f.dialAndCreateClient(sshConfig, net.JoinHostPort(localServer.Address, fmt.Sprintf("%d", localServer.Port)))
			if errClient != nil {
				slog.Error("failed to connect to remote host", log.ErrAttr(errClient))

				return
			}

			defer func() {
				if errClose := scpClient.Close(); errClose != nil {
					slog.Error("failed to close scp client", log.ErrAttr(errClose))
				}
			}()

			if err := f.onConnect(ctx, scpClient, localServer); err != nil {
				slog.Error("onConnect function errored", log.ErrAttr(err))
			}
		}(server)
	}

	return nil
}

// dialAndCreateSession connects to the remote server with the config. client.Close must be called.
func (f SCPExecer) dialAndCreateClient(sshConfig domain.ConfigSSH, address string) (storage.Storager, error) {
	clientConfig, errConfig := f.createConfig(sshConfig)
	if errConfig != nil {
		return nil, errConfig
	}

	client, errClient := scp.NewStorager(address, sshConfig.Timeout, clientConfig)
	if errClient != nil {
		return nil, errors.Join(errClient, errConnect)
	}

	return client, nil
}

func (f SCPExecer) createConfig(config domain.ConfigSSH) (*ssh.ClientConfig, error) {
	if config.Username == "" {
		return nil, errUsername
	}

	var authMethod []ssh.AuthMethod

	switch {
	case config.Password == "" && config.PrivateKeyPath == "":
		return nil, errInsufficientAuthMethod
	case config.PrivateKeyPath == "":
		authMethod = append(authMethod, ssh.Password(config.Password))
	default:
		signer, errSigner := f.createSignerFromKey(config)
		if errSigner != nil {
			return nil, errSigner
		}

		authMethod = append(authMethod, ssh.PublicKeys(signer))
	}

	sshClientConfig := &ssh.ClientConfig{
		User:            config.Username,
		Auth:            authMethod,
		HostKeyCallback: ssh.InsecureIgnoreHostKey(), //nolint:gosec
		Timeout:         config.Timeout,
	}

	return sshClientConfig, nil
}

func (f SCPExecer) createSignerFromKey(config domain.ConfigSSH) (ssh.Signer, error) {
	keyBytes, err := os.ReadFile(config.PrivateKeyPath)
	if err != nil {
		return nil, errors.Join(err, errReadPrivateKey)
	}

	if config.Password != "" {
		key, errParse := ssh.ParsePrivateKeyWithPassphrase(keyBytes, []byte("password"))
		if errParse != nil {
			return nil, errors.Join(errParse, errParsePrivateKey)
		}

		return key, nil
	}

	key, errParse := ssh.ParsePrivateKey(keyBytes)
	if errParse != nil {
		return nil, errors.Join(errParse, errParsePrivateKey)
	}

	return key, nil
}
