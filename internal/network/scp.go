package network

import (
	"context"
	"encoding/base64"
	"errors"
	"log/slog"
	"net"
	"os"
	"strconv"
	"sync"
	"time"

	"github.com/leighmacdonald/gbans/internal/database"
	"github.com/leighmacdonald/gbans/internal/domain"
	"github.com/leighmacdonald/gbans/pkg/log"
	"github.com/mitchellh/go-homedir"
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
	errHomeDir                = errors.New("failed to expand home dir")
	errKeyVerificationFailed  = errors.New("host key validation failed")
)

// OnClientConnect is called once a successful ssh connection is established.
type OnClientConnect func(ctx context.Context, client storage.Storager, server []domain.Server) error

// SCPExecer can be used to execute scp (ssh) operations on a remote host. It connects to all configured active
// servers and will execute the provided OnClientConnect function once connected. It's up to the caller
// to implement this function and handle any required functionality within it. Caller does not need to close the
// connection.
type SCPExecer struct {
	serversUsecase domain.ServersUsecase
	database       database.Database
	configUsecase  domain.ConfigUsecase
	onConnect      OnClientConnect
}

func NewSCPExecer(database database.Database, configUsecase domain.ConfigUsecase, serversUsecase domain.ServersUsecase, onConnect OnClientConnect) SCPExecer {
	return SCPExecer{
		database:       database,
		configUsecase:  configUsecase,
		serversUsecase: serversUsecase,
		onConnect:      onConnect,
	}
}

func (f SCPExecer) Start(ctx context.Context) {
	updateTimer := time.NewTimer(time.Minute * 5)

	if errUpdate := f.update(ctx); errUpdate != nil {
		slog.Error("Error querying ssh demo", log.ErrAttr(errUpdate))
	}

	for {
		select {
		case <-updateTimer.C:
			if errUpdate := f.update(ctx); errUpdate != nil {
				slog.Error("Error querying ssh demo", log.ErrAttr(errUpdate))
			}

			updateTimer.Reset(time.Minute * 5)
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

	// Since multiple instances can exist on a single host we map common servers to a single host address and
	// perform all operations using a single connection to the host.
	mappedServers := map[string][]domain.Server{}

	for _, server := range servers {
		_, ok := mappedServers[server.Address]
		if !ok {
			mappedServers[server.Address] = []domain.Server{}
		}

		mappedServers[server.Address] = append(mappedServers[server.Address], server)
	}

	sshConfig := f.configUsecase.Config().SSH

	waitGroup := &sync.WaitGroup{}

	for address := range mappedServers {
		waitGroup.Add(1)

		go f.updateServer(ctx, waitGroup, address, mappedServers[address], sshConfig)
	}

	waitGroup.Wait()

	return nil
}

func (f SCPExecer) updateServer(ctx context.Context, waitGroup *sync.WaitGroup, addr string, addrServers []domain.Server, sshConfig domain.ConfigSSH) {
	defer waitGroup.Done()

	scpClient, errClient := f.configAndDialClient(ctx, sshConfig, net.JoinHostPort(addr, strconv.Itoa(sshConfig.Port)))
	if errClient != nil {
		slog.Error("failed to connect to remote host", log.ErrAttr(errClient))

		return
	}

	defer func() {
		if errClose := scpClient.Close(); errClose != nil {
			slog.Error("failed to close scp client", log.ErrAttr(errClose))
		}
	}()

	if err := f.onConnect(ctx, scpClient, addrServers); err != nil {
		slog.Error("onConnect function errored", log.ErrAttr(err))
	}
}

// configAndDialClient connects to the remote server with the config. client.Close must be called.
func (f SCPExecer) configAndDialClient(ctx context.Context, sshConfig domain.ConfigSSH, address string) (storage.Storager, error) {
	clientConfig, errConfig := f.createConfig(ctx, sshConfig)
	if errConfig != nil {
		return nil, errConfig
	}

	client, errClient := scp.NewStorager(address, time.Duration(sshConfig.Timeout)*time.Second, clientConfig)
	if errClient != nil {
		return nil, errors.Join(errClient, errConnect)
	}

	return client, nil
}

func (f SCPExecer) createConfig(ctx context.Context, config domain.ConfigSSH) (*ssh.ClientConfig, error) {
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
		HostKeyCallback: f.trustedHostKeyCallback(ctx),
		Timeout:         time.Duration(config.Timeout) * time.Second,
	}

	return sshClientConfig, nil
}

func (f SCPExecer) createSignerFromKey(config domain.ConfigSSH) (ssh.Signer, error) {
	fullPath, errPath := homedir.Expand(config.PrivateKeyPath)
	if errPath != nil {
		return nil, errors.Join(errPath, errHomeDir)
	}

	keyBytes, err := os.ReadFile(fullPath)
	if err != nil {
		return nil, errors.Join(err, errReadPrivateKey)
	}

	if config.Password != "" {
		key, errParse := ssh.ParsePrivateKeyWithPassphrase(keyBytes, []byte(config.Password))
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

// KeyString generates ecdsa-sha2-nistp256 AAAAE2VjZHNhLXNoYTItbmlzdHAyNTY.... from a public key.
func keyString(k ssh.PublicKey) string {
	return k.Type() + " " + base64.StdEncoding.EncodeToString(k.Marshal())
}

// trustedHostKeyCallback handles validation of the host key. If a host key is not already
// known it is automatically stored in the database as the trusted key on the first connection.
// Subsequent connections will require the same key or be rejected. If you want to skip the auto
// trust of the first key seen, you must insert the host keys into the database manually into the
// host_key table.
func (f SCPExecer) trustedHostKeyCallback(ctx context.Context) ssh.HostKeyCallback {
	getKey := func(addr string) (string, error) {
		var key string

		if errRow := f.database.
			QueryRow(ctx, `SELECT key FROM host_key WHERE address = $1`, addr).
			Scan(&key); errRow != nil {
			return "", f.database.DBErr(errRow)
		}

		return key, nil
	}

	setKey := func(addr string, key string) error {
		const query = `INSERT INTO host_key (address, key, created_on) VALUES ($1, $2, $3)`
		if err := f.database.Exec(ctx, query, addr, key, time.Now()); err != nil {
			return f.database.DBErr(err)
		}

		return nil
	}

	return func(hostname string, addr net.Addr, pubKey ssh.PublicKey) error {
		slog.Debug("SSH Connect", slog.String("hostname", hostname), slog.String("addr", addr.String()))

		trustedPubKeyString, errKey := getKey(addr.String())
		if errKey != nil && !errors.Is(errKey, domain.ErrNoResult) {
			return errKey
		}

		pubKeyString := keyString(pubKey)

		if trustedPubKeyString == "" {
			if errSet := setKey(addr.String(), pubKeyString); errSet != nil {
				return errSet
			}
		}

		if trustedPubKeyString != pubKeyString {
			slog.Error("Host key validation failed", slog.String("hostname", hostname))

			return errKeyVerificationFailed
		}

		return nil
	}
}
