package circuit

import (
	"bufio"
	"context"
	"encoding/hex"
	"fmt"
	"net"
	"os"
	"strings"
)

type Client struct {
	addr       string
	cookiePath string
	conn       net.Conn
	reader     *bufio.Reader
}

func NewClient(addr string, cookiePath string) *Client {
	return &Client{
		addr:       addr,
		cookiePath: cookiePath,
	}
}

func (c *Client) Connect(ctx context.Context) error {
	dialer := net.Dialer{}
	conn, err := dialer.DialContext(ctx, "tcp", c.addr)
	if err != nil {
		return fmt.Errorf("Connect: %w", err)
	}
	c.conn = conn
	c.reader = bufio.NewReader(conn)
	return nil
}

func (c *Client) Authenticate(ctx context.Context) error {
	cookieBytes, err := os.ReadFile(c.cookiePath)
	if err != nil {
		return fmt.Errorf("Authenticate: read cookie: %w", err)
	}

	hexCookie := hex.EncodeToString(cookieBytes)
	cmd := fmt.Sprintf("AUTHENTICATE %s\r\n", hexCookie)

	if err := c.send(cmd); err != nil {
		return fmt.Errorf("Authenticate: send: %w", err)
	}

	resp, err := c.readLine()
	if err != nil {
		return fmt.Errorf("Authenticate: read response: %w", err)
	}

	if !strings.HasPrefix(resp, "250") {
		return fmt.Errorf("Authenticate: unexpected response: %s", resp)
	}

	return nil
}

func (c *Client) SignalNewnym(ctx context.Context) error {
	if err := c.send("SIGNAL NEWNYM\r\n"); err != nil {
		return fmt.Errorf("SignalNewnym: send: %w", err)
	}

	resp, err := c.readLine()
	if err != nil {
		return fmt.Errorf("SignalNewnym: read response: %w", err)
	}

	if !strings.HasPrefix(resp, "250") {
		return fmt.Errorf("SignalNewnym: unexpected response: %s", resp)
	}

	return nil
}

func (c *Client) Close() error {
	if c.conn == nil {
		return nil
	}
	err := c.conn.Close()
	c.conn = nil
	c.reader = nil
	return err
}

func (c *Client) send(cmd string) error {
	_, err := fmt.Fprint(c.conn, cmd)
	return err
}

func (c *Client) readLine() (string, error) {
	line, err := c.reader.ReadString('\n')
	if err != nil {
		return "", err
	}
	return strings.TrimRight(line, "\r\n"), nil
}

func BuildAuthCommand(cookieBytes []byte) string {
	return fmt.Sprintf("AUTHENTICATE %s\r\n", hex.EncodeToString(cookieBytes))
}

func ParseResponse(line string) (code string, message string, err error) {
	line = strings.TrimRight(line, "\r\n")
	if len(line) < 4 {
		return "", "", fmt.Errorf("ParseResponse: response too short: %q", line)
	}

	code = line[:3]
	message = strings.TrimSpace(line[3:])
	return code, message, nil
}

func BuildNewnymCommand() string {
	return "SIGNAL NEWNYM\r\n"
}
