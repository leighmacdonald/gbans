package scp

import (
	"context"
	"encoding/base64"
	"errors"
	"log/slog"
	"net"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/leighmacdonald/gbans/internal/config"
	"github.com/leighmacdonald/gbans/internal/database"
	"github.com/leighmacdonald/gbans/pkg/log"
	"github.com/mitchellh/go-homedir"
	"github.com/viant/afs/scp"
	"github.com/viant/afs/storage"
	"golang.org/x/crypto/ssh"
)

// KeyStore is responsible for storing and retrieving host keys
type KeyStore interface {
	Set(ctx context.Context, host string, key string) error
	Get(ctx context.Context, host string) error
}

var (
	errInsufficientAuthMethod = errors.New("ssh password or private key missing")
	errReadPrivateKey         = errors.New("failed to read private key contents")
	errParsePrivateKey        = errors.New("failed to parse private key")
	errUsername               = errors.New("invalid username")
	errConnect                = errors.New("failed to connect to ssh server")
	errHomeDir                = errors.New("failed to expand home dir")
	errKeyVerificationFailed  = errors.New("host key validation failed")
	ErrInvalidAddress         = errors.New("invalid address")
)

type ConnectionHandler interface {
	Handler(ctx context.Context, client storage.Storager, server ServerInfo) error
}

// SCPHandler can be used to execute scp (ssh) operations on a remote host. It connects to all configured active
// servers and will execute the provided OnClientConnect function once connected. It's up to the caller
// to implement this function and handle any required functionality within it. Caller does not need to close the
// connection.
type SCPHandler struct {
	details  ServerInfo
	config   *config.Configuration
	handlers []ConnectionHandler
	keyStore KeyStore
	conn     storage.Storager
}

func NewSCPHandler(database Repository, config *config.Configuration) SCPHandler {
	return SCPHandler{config: config}
}

func (f SCPHandler) Address() string {
	return f.details.Address
}

func (f SCPHandler) Close() {
	if errClose := f.conn.Close(); errClose != nil {
		slog.Error("failed to close scp client", log.ErrAttr(errClose))
	}
}

func (f SCPHandler) Update(ctx context.Context) error {
	// Since multiple instances can exist on a single host we map common servers to a single host address and
	// perform all operations using a single connection to the host.
	mappedServers := map[string][]ServerInfo{}

	for _, server := range servers {

		actualAddr := HostPart(server.Address)

		mappedServers[actualAddr] = append(mappedServers[actualAddr], server)
	}

	sshConfig := f.config.Config().SSH
	waitGroup := &sync.WaitGroup{}

	for address := range mappedServers {
		waitGroup.Go(func() {
			f.updateServer(ctx, waitGroup, address, mappedServers[address], sshConfig)
		})
	}

	waitGroup.Wait()

	return nil
}

func (f SCPHandler) connect(ctx context.Context, sshConfig config.SSH) error {
	if f.conn != nil {
		return nil
	}

	client, errClient := f.configAndDialClient(ctx, sshConfig, net.JoinHostPort(f.details.Address, strconv.Itoa(sshConfig.Port)))
	if errClient != nil {
		return errClient
	}

	f.conn = client
}

// configAndDialClient connects to the remote server with the config. client.Close must be called.
func (f SCPHandler) configAndDialClient(ctx context.Context, sshConfig config.SSH, address string) (storage.Storager, error) {
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

func (f SCPHandler) createConfig(ctx context.Context, config config.SSH) (*ssh.ClientConfig, error) {
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
		HostKeyCallback: f.trustedHostKeyCallback,
		Timeout:         time.Duration(config.Timeout) * time.Second,
	}

	return sshClientConfig, nil
}

func (f SCPHandler) createSignerFromKey(config config.SSH) (ssh.Signer, error) {
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
func (f SCPHandler) trustedHostKeyCallback(hostname string, addr net.Addr, pubKey ssh.PublicKey) error {
	slog.Debug("SSH Connect", slog.String("hostname", hostname), slog.String("addr", addr.String()))

	trustedPubKeyString, errKey := f.repo.getKey(context.Background(), addr.String())
	if errKey != nil && !errors.Is(errKey, database.ErrNoResult) {
		return errKey
	}

	pubKeyString := keyString(pubKey)

	if trustedPubKeyString == "" {
		if errSet := f.repo.setKey(context.Background(), addr.String(), pubKeyString); errSet != nil {
			return errSet
		}

		trustedPubKeyString = pubKeyString
	}

	if trustedPubKeyString != pubKeyString {
		slog.Error("Host key validation failed", slog.String("hostname", hostname))

		return errKeyVerificationFailed
	}

	return nil
}

func HostPart(address string) string {
	parts := strings.Split(address, ":")
	if len(parts) != 2 {
		return address
	}
	return parts[0]
}
