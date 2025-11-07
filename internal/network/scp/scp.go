package scp

import (
	"context"
	"encoding/base64"
	"errors"
	"log/slog"
	"net"
	"os"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/leighmacdonald/gbans/internal/config"
	"github.com/leighmacdonald/gbans/internal/database"
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
	ErrInvalidAddress         = errors.New("invalid address")
)

// KeyStore is responsible for storing and retrieving host keys.
type KeyStore interface {
	SetHostKey(ctx context.Context, host string, key string) error
	GetHostKey(ctx context.Context, host string) (string, error)
}

type ConnectionHandler interface {
	DownloadHandler(ctx context.Context, client storage.Storager, server ServerInfo) error
}

// Connection can be used to execute scp (ssh) operations on a remote host. It connects to all configured active
// servers and will execute the provided OnClientConnect function once connected. It's up to the caller
// to implement this function and handle any required functionality within it. Caller does not need to close the
// connection.
type Connection struct {
	details  ServerInfo
	handlers []ConnectionHandler
	repo     KeyStore
	conn     storage.Storager
	config   config.SSH
}

func NewConnection(database KeyStore, config config.SSH) Connection {
	return Connection{repo: database, config: config}
}

func (f *Connection) Address() string {
	return f.details.Address
}

func (f *Connection) Close() {
	if f.conn == nil {
		return
	}

	if errClose := f.conn.Close(); errClose != nil {
		slog.Error("failed to close scp client", slog.String("error", errClose.Error()))
	}
}

func (f *Connection) AddHandler(handler ConnectionHandler) {
	if slices.Contains(f.handlers, handler) {
		return
	}

	f.handlers = append(f.handlers, handler)
}

func (f *Connection) Update(ctx context.Context) error {
	if errConnect := f.connect(); errConnect != nil {
		return errConnect
	}

	var errs error
	for _, handler := range f.handlers {
		if errHandler := handler.DownloadHandler(ctx, f.conn, f.details); errHandler != nil {
			errs = errors.Join(errHandler)
		}
	}

	return errs
}

func (f *Connection) connect() error {
	if f.conn != nil {
		return nil
	}

	client, errClient := configAndDialClient(f.repo, f.config, net.JoinHostPort(f.details.Address, strconv.Itoa(f.config.Port)))
	if errClient != nil {
		return errClient
	}

	f.conn = client

	return nil
}

// configAndDialClient connects to the remote server with the config. client.Close must be called.
func configAndDialClient(repo KeyStore, sshConfig config.SSH, address string) (storage.Storager, error) { //nolint:ireturn
	clientConfig, errConfig := createConfig(repo, sshConfig)
	if errConfig != nil {
		return nil, errConfig
	}

	client, errClient := scp.NewStorager(address, time.Duration(sshConfig.Timeout)*time.Second, clientConfig)
	if errClient != nil {
		return nil, errors.Join(errClient, errConnect)
	}

	return client, nil
}

func createConfig(repo KeyStore, config config.SSH) (*ssh.ClientConfig, error) {
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
		signer, errSigner := createSignerFromKey(config)
		if errSigner != nil {
			return nil, errSigner
		}

		authMethod = append(authMethod, ssh.PublicKeys(signer))
	}

	return &ssh.ClientConfig{
		User:            config.Username,
		Auth:            authMethod,
		HostKeyCallback: trustedHostKeyCallback(repo),
		Timeout:         time.Duration(config.Timeout) * time.Second,
	}, nil
}

func createSignerFromKey(config config.SSH) (ssh.Signer, error) { //nolint:ireturn
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

// trustedHostKeyCallback handles validation of the host key. If a host key is not already
// known it is automatically stored in the database as the trusted key on the first connection.
// Subsequent connections will require the same key or be rejected. If you want to skip the auto
// trust of the first key seen, you must insert the host keys into the database manually into the
// host_key table.
func trustedHostKeyCallback(repo KeyStore) func(hostname string, addr net.Addr, pubKey ssh.PublicKey) error {
	return func(hostname string, addr net.Addr, pubKey ssh.PublicKey) error {
		slog.Debug("SSH Connect", slog.String("hostname", hostname), slog.String("addr", addr.String()))

		trustedPubKeyString, errKey := repo.GetHostKey(context.Background(), addr.String())
		if errKey != nil && !errors.Is(errKey, database.ErrNoResult) {
			return errKey
		}

		pubKeyString := keyString(pubKey)

		if trustedPubKeyString == "" {
			if errSet := repo.SetHostKey(context.Background(), addr.String(), pubKeyString); errSet != nil {
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
}

// KeyString generates ecdsa-sha2-nistp256 AAAAE2VjZHNhLXNoYTItbmlzdHAyNTY.... from a public key.
func keyString(k ssh.PublicKey) string {
	return k.Type() + " " + base64.StdEncoding.EncodeToString(k.Marshal())
}

func HostPart(address string) string {
	parts := strings.Split(address, ":")
	if len(parts) != 2 {
		return address
	}

	return parts[0]
}
