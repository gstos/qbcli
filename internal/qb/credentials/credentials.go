package credentials

import (
	"fmt"
	"golang.org/x/crypto/scrypt"
	"net/url"
	"strconv"
)

type Credentials struct {
	Scheme   string
	Host     string
	Port     int
	Username string
	Password string
}

type Option func(*Credentials) error

func defaultOptions() []Option {
	return []Option{}
}

const defaultScheme = "http"
const defaultHost = "127.0.0.1"
const defaultPort = 8080

func New(username, password string, opts ...Option) *Credentials {
	creds := &Credentials{
		Scheme:   defaultScheme,
		Host:     defaultHost,
		Port:     defaultPort,
		Username: username,
		Password: password,
	}
	opts = append(defaultOptions(), opts...)
	for _, opt := range opts {
		err := opt(creds)
		if err != nil {
			panic(err)
		}
	}
	return creds
}

func FromURL(hostURL *url.URL, opts ...Option) (*Credentials, error) {
	creds := &Credentials{}

	if hostName, hostPort, err := parseHost(hostURL); err != nil {
		return nil, err
	} else {
		creds.Host = hostName
		creds.Port = hostPort
	}

	if hostUser, hostPassword, hasUser := parseUserPassword(hostURL); hasUser {
		creds.Username = hostUser
		creds.Password = hostPassword
	}

	if scheme, err := parseScheme(hostURL); err != nil {
		return nil, err
	} else {
		creds.Scheme = scheme
	}

	for _, opt := range opts {
		err := opt(creds)
		if err != nil {
			return nil, err
		}
	}

	if err := creds.Validate(); err != nil {
		return nil, err
	}
	return creds, nil
}

func WithScheme(scheme string) Option {
	return func(creds *Credentials) error {
		if scheme != "http" && scheme != "https" {
			return fmt.Errorf("invalid scheme: %s", scheme)
		}
		creds.Scheme = scheme
		return nil
	}
}

func WithHost(host string) Option {
	return func(creds *Credentials) error {
		creds.Host = host
		return nil
	}
}

func WithPort(port int) Option {
	return func(creds *Credentials) error {
		if port < 0 || port > 65535 {
			return fmt.Errorf("invalid port: %d", port)
		}
		creds.Port = port
		return nil
	}
}

func WithUsername(username string) Option {
	return func(creds *Credentials) error {
		creds.Username = username
		return nil
	}
}

func WithPassword(password string) Option {
	return func(creds *Credentials) error {
		creds.Password = password
		return nil
	}
}

func (creds *Credentials) String() string {
	return fmt.Sprintf("%s://%s:<password>@%s:%d", creds.Scheme, creds.Username, creds.Host, creds.Port)
}

func (creds *Credentials) DeriveBaseURL() string {
	return fmt.Sprintf("%s://%s:%d", creds.Scheme, creds.Host, creds.Port)
}

func (creds *Credentials) DeriveFileName() string {
	return fmt.Sprintf("%s-%s__at__%s-%d", creds.Scheme, creds.Username, creds.Host, creds.Port)
}

func (creds *Credentials) DeriveKey(keyLen int, salt []byte) ([]byte, error) {
	password := []byte(creds.Password)
	return scrypt.Key(password, salt, 1<<15, 8, 1, keyLen)
}

func (creds *Credentials) Validate() error {
	if creds.Scheme != "http" && creds.Scheme != "https" {
		return fmt.Errorf("invalid scheme: %s", creds.Scheme)
	}

	if creds.Host == "" {
		return fmt.Errorf("empty host")
	}

	if creds.Port < 0 || creds.Port > 65535 {
		return fmt.Errorf("invalid port: %d", creds.Port)
	}

	if creds.Username == "" {
		return fmt.Errorf("empty username")
	}
	return nil
}

func parseScheme(rawURL *url.URL) (string, error) {
	if rawURL.Scheme == "" {
		return defaultScheme, nil
	}

	if rawURL.Scheme != "http" && rawURL.Scheme != "https" {
		return "", fmt.Errorf("invalid scheme: %s", rawURL.Scheme)
	}

	return rawURL.Scheme, nil
}

func parseHost(rawURL *url.URL) (string, int, error) {
	if rawURL.Host == "" {
		return defaultHost, defaultPort, nil
	}

	hostName := rawURL.Hostname()
	portStr := rawURL.Port()

	if portStr == "" {
		return hostName, defaultPort, nil
	}

	hostPort, err := strconv.Atoi(portStr)
	if err != nil {
		return "", 0, fmt.Errorf("invalid port number: %s", portStr)
	}

	if hostPort < 0 || hostPort > 65535 {
		return "", 0, fmt.Errorf("invalid port: %d", hostPort)
	}
	return hostName, hostPort, nil
}

func parseUserPassword(rawURL *url.URL) (string, string, bool) {
	if rawURL.User == nil {
		return "", "", false
	}

	username := rawURL.User.Username()
	password, _ := rawURL.User.Password()

	return username, password, true
}
