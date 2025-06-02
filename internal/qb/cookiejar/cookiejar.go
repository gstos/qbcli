package cookiejar

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/gstos/qbcli/internal/qb/credentials"
)

const (
	defaultSaltSize  = 16
	defaultNonceSize = 12
	defaultKeyLen    = 32 // AES-256
)

type Option func(*CookieJar)

type CookieJar struct {
	Dir       string
	saltSize  int
	nonceSize int
	keyLen    int
	log       *slog.Logger
	LogLevel  *slog.LevelVar
}

type EncryptedCookieFile struct {
	ExpiresAt time.Time `json:"expiresAt:omitempty"`
	CipherB64 string    `json:"cookie"` // base64(salt + nonce + ciphertext)
}

func defaultOptions() []Option {
	return []Option{
		WithNonceSize(defaultNonceSize),
		WithKeyLen(defaultKeyLen),
		WithSaltSize(defaultSaltSize),
	}
}

func New(dir string, opts ...Option) *CookieJar {
	jar := &CookieJar{
		Dir: dir,
		log: slog.New(slog.DiscardHandler),
	}

	opts = append(defaultOptions(), opts...)
	for _, opt := range opts {
		opt(jar)
	}

	return jar
}

func WithSaltSize(saltSize int) Option {
	return func(jar *CookieJar) {
		jar.saltSize = saltSize
	}
}

func WithNonceSize(nonceSize int) Option {
	return func(jar *CookieJar) {
		jar.nonceSize = nonceSize
	}
}

func WithKeyLen(keyLen int) Option {
	return func(jar *CookieJar) {
		jar.keyLen = keyLen
	}
}

func WithLogger(logger *slog.Logger) Option {
	return func(cli *CookieJar) {
		cli.log = logger
	}
}

func WithCustomLogger(w io.Writer, opts *slog.HandlerOptions) Option {
	return func(jar *CookieJar) {
		if opts == nil {
			opts = &slog.HandlerOptions{}
		}

		if opts.Level == nil {
			opts.Level = jar.LogLevel
		}

		jar.log = slog.New(slog.NewTextHandler(w, opts))
	}
}

func WithLogLevel(logLevel slog.Level) Option {
	return func(jar *CookieJar) {
		jar.log = slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: jar.LogLevel}))
		jar.LogLevel.Set(logLevel)
	}
}

func (jar *CookieJar) DeriveFileName(creds *credentials.Credentials) string {
	fileName := creds.DeriveFileName()
	return fmt.Sprintf("%s.cookie", fileName)
}

func (jar *CookieJar) DerivePath(creds *credentials.Credentials) string {
	return filepath.Join(jar.Dir, jar.DeriveFileName(creds))
}

func (jar *CookieJar) Store(creds *credentials.Credentials, cookie *http.Cookie) error {
	filePath := jar.DerivePath(creds)

	log := jar.log.With("path", filePath)

	if err := os.MkdirAll(jar.Dir, 0o700); err != nil {
		log.Error("creating cache dir", "path", jar.Dir, "error", err)
		return fmt.Errorf("creating cache dir: %w", err)
	}

	salt := make([]byte, jar.saltSize)
	if _, err := rand.Read(salt); err != nil {
		log.Error("generating salt", "error", err)
		return fmt.Errorf("generating salt: %w", err)
	}
	key, err := creds.DeriveKey(jar.keyLen, salt)
	if err != nil {
		log.Error("deriving key", "error", err)
		return fmt.Errorf("deriving key: %w", err)
	}

	plaintext, err := json.Marshal(cookie)
	if err != nil {
		log.Error("encoding cookie", "error", err)
		return fmt.Errorf("encoding cookie: %w", err)
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		log.Error("creating cipher", "error", err)
		return fmt.Errorf("creating cipher: %w", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		log.Error("creating GCM", "error", err)
		return fmt.Errorf("creating GCM: %w", err)
	}

	nonce := make([]byte, jar.nonceSize)
	if _, err := rand.Read(nonce); err != nil {
		log.Error("generating nonce", "error", err)
		return fmt.Errorf("generating nonce: %w", err)
	}

	ciphertext := gcm.Seal(nil, nonce, plaintext, nil)
	combined := append(salt, append(nonce, ciphertext...)...)

	entry := EncryptedCookieFile{
		ExpiresAt: cookie.Expires,
		CipherB64: base64.StdEncoding.EncodeToString(combined),
	}

	data, err := json.MarshalIndent(entry, "", "  ")
	if err != nil {
		log.Error("serializing encrypted cookie", "error", err)
		return fmt.Errorf("serializing encrypted cookie: %w", err)
	}

	err = os.WriteFile(filePath, data, 0o600)
	if err != nil {
		jar.log.Error("writing cookie file", "path", filePath, "error", err)
		return fmt.Errorf("writing cookie file: %w", err)
	}

	return nil
}

func (jar *CookieJar) Retrieve(creds *credentials.Credentials) (*http.Cookie, error) {
	filePath := jar.DerivePath(creds)
	log := jar.log.With("path", filePath)

	data, err := os.ReadFile(filePath)
	if err != nil {
		log.Debug("reading cookie file", "path", filePath, "error", err)
		return nil, fmt.Errorf("reading cookie file: %w", err)
	}

	var entry EncryptedCookieFile
	if err := json.Unmarshal(data, &entry); err != nil {
		log.Error("parsing cookie metadata", "error", err)
		jar.cleanUpCookieFile(filePath)
		return nil, fmt.Errorf("failed to parse cookie metadata: %w", err)
	}

	if !entry.ExpiresAt.IsZero() && time.Now().After(entry.ExpiresAt) {
		log.Debug("cookie expired", "path", filePath, "expiresAt", entry.ExpiresAt)
		jar.cleanUpCookieFile(filePath)
		return nil, errors.New("cookie expired")
	}

	decoded, err := base64.StdEncoding.DecodeString(entry.CipherB64)
	if err != nil {
		log.Error("decoding cookie: invalid base64 encoding", "error", err)
		jar.cleanUpCookieFile(filePath)
		return nil, fmt.Errorf("invalid base64 encoding: %w", err)
	}
	if len(decoded) < jar.saltSize+jar.nonceSize {
		log.Error("decoding cookie: payload too short", "error", err)
		jar.cleanUpCookieFile(filePath)
		return nil, errors.New("encrypted payload too short")
	}

	salt := decoded[:jar.saltSize]
	nonce := decoded[jar.saltSize : jar.saltSize+jar.nonceSize]
	ciphertext := decoded[jar.saltSize+jar.nonceSize:]

	key, err := creds.DeriveKey(jar.keyLen, salt)
	if err != nil {
		log.Error("decoding cookie: failed to derive key", "error", err)
		jar.cleanUpCookieFile(filePath)
		return nil, fmt.Errorf("failed to derive key: %w", err)
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		log.Error("decoding cookie: failed to create cipher", "error", err)
		jar.cleanUpCookieFile(filePath)
		return nil, fmt.Errorf("failed to create cipher: %w", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		log.Error("decoding cookie: failed to create GCM", "error", err)
		jar.cleanUpCookieFile(filePath)
		return nil, fmt.Errorf("failed to create GCM: %w", err)
	}

	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		log.Error("decoding cookie: decryption failed", "error", err)
		jar.cleanUpCookieFile(filePath)
		return nil, fmt.Errorf("decryption failed: %w", err)
	}

	var cookie http.Cookie
	if err := json.Unmarshal(plaintext, &cookie); err != nil {
		log.Error("decoding cookie: failed to parse cookie JSON", "error", err)
		jar.cleanUpCookieFile(filePath)
		return nil, fmt.Errorf("failed to parse cookie JSON: %w", err)
	}

	if IsExpired(&cookie) {
		jar.cleanUpCookieFile(filePath)
		return nil, fmt.Errorf("cookie expired: %s", cookie.Expires.Format(time.RFC3339))
	}

	return &cookie, nil
}

func (jar *CookieJar) Delete(creds *credentials.Credentials) error {
	filePath := jar.DerivePath(creds)
	if err := os.Remove(filePath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to delete cookie: %w", err)
	}
	return nil
}

func (jar *CookieJar) cleanUpCookieFile(filePath string) {
	if err := os.Remove(filePath); err != nil && !os.IsNotExist(err) {
		jar.log.Error("deleting cookie file", "path", filePath, "error", err)
	}
}

func IsExpired(c *http.Cookie) bool {
	if c.Expires.IsZero() {
		return false
	}
	return time.Now().After(c.Expires)
}
