// Package nut provides a Golang interface for interacting with Network UPS Tools (NUT).
//
// It communicates with NUT over the TCP protocol
package nut

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"net"
	"strings"
	"time"
)

// Client contains information about the NUT server as well as the connection.
type Client struct {
	Version         string
	ProtocolVersion string
	Hostname        net.Addr
	conn            net.Conn
	ctx             context.Context
	stop            func() bool
}

// Connect accepts a hostname/IP string and an optional port, then creates a connection to NUT, returning a Client.
func Connect(hostname string, _port ...int) (Client, error) {
	return ConnectContext(context.Background(), hostname, _port...)
}

// ConnectContext connects to NUT and verifies VER and NETVER.
// The context controls dialing, the handshake, and all subsequent I/O.
// Call Close or Disconnect when finished, even when ctx has no deadline.
func ConnectContext(ctx context.Context, hostname string, _port ...int) (Client, error) {
	port := 3493
	if len(_port) > 0 {
		port = _port[0]
	}
	conn, err := (&net.Dialer{}).DialContext(ctx, "tcp", net.JoinHostPort(hostname, fmt.Sprint(port)))
	if err != nil {
		return Client{}, err
	}
	client := Client{Hostname: conn.RemoteAddr(), conn: conn, ctx: ctx}
	client.stop = context.AfterFunc(ctx, func() { _ = conn.Close() })
	if _, err = client.GetVersion(); err == nil {
		_, err = client.GetNetworkProtocolVersion()
	}
	if err != nil {
		_ = client.Close()
		if ctx.Err() != nil {
			return Client{}, ctx.Err()
		}
		if deadline, ok := ctx.Deadline(); ok && !time.Now().Before(deadline) {
			return Client{}, context.DeadlineExceeded
		}
		return Client{}, err
	}
	return client, nil
}

// Close releases the connection without sending LOGOUT. Repeated calls are safe.
func (c *Client) Close() error {
	if c.stop != nil {
		c.stop()
	}
	if c.conn == nil {
		return nil
	}
	err := c.conn.Close()
	if errors.Is(err, net.ErrClosed) {
		return nil
	}
	return err
}

// Disconnect sends LOGOUT and closes the connection, even if LOGOUT fails.
func (c *Client) Disconnect() (bool, error) {
	defer c.Close()
	logoutResp, err := c.SendCommand("LOGOUT")
	if err != nil {
		return false, err
	}
	if logoutResp[0] == "OK Goodbye" || logoutResp[0] == "Goodbye..." {
		return true, nil
	}
	return false, nil
}

func (c *Client) operationDeadline() time.Time {
	deadline := time.Now().Add(2 * time.Second)
	if c.ctx != nil {
		if overall, ok := c.ctx.Deadline(); ok && overall.Before(deadline) {
			deadline = overall
		}
	}
	return deadline
}

// ReadResponse is a convenience function for reading newline delimited responses.
func (c *Client) ReadResponse(endLine string, multiLineResponse bool) (resp []string, err error) {
	if err := c.conn.SetReadDeadline(c.operationDeadline()); err != nil {
		return nil, err
	}
	connbuff := bufio.NewReader(c.conn)
	response := []string{}

	for {
		line, err := connbuff.ReadString('\n')
		if err != nil {
			return nil, fmt.Errorf("error reading response: %v", err)
		}
		if len(line) > 0 {
			cleanLine := strings.TrimSuffix(line, "\n")
			lines := strings.Split(cleanLine, "\n")
			response = append(response, lines...)
			if line == endLine || multiLineResponse == false {
				break
			}
		}
	}

	return response, err
}

// SendCommand sends the string cmd to the device, and returns the response.
func (c *Client) SendCommand(cmd string) (resp []string, err error) {
	cmd = fmt.Sprintf("%v\n", cmd)
	endLine := fmt.Sprintf("END %s", cmd)
	if strings.HasPrefix(cmd, "USERNAME ") || strings.HasPrefix(cmd, "PASSWORD ") || strings.HasPrefix(cmd, "SET ") || strings.HasPrefix(cmd, "HELP ") || strings.HasPrefix(cmd, "VER ") || strings.HasPrefix(cmd, "NETVER ") {
		endLine = "OK\n"
	}
	if c.ctx != nil && c.ctx.Err() != nil {
		return nil, c.ctx.Err()
	}
	if err := c.conn.SetWriteDeadline(c.operationDeadline()); err != nil {
		return nil, err
	}
	_, err = fmt.Fprint(c.conn, cmd)
	if err != nil {
		return []string{}, err
	}

	resp, err = c.ReadResponse(endLine, strings.HasPrefix(cmd, "LIST "))
	if err != nil {
		return []string{}, err
	}

	if strings.HasPrefix(resp[0], "ERR ") {
		return []string{}, errorForMessage(strings.Split(resp[0], " ")[1])
	}

	return resp, nil
}

// Authenticate accepts a username and passwords and uses them to authenticate the existing NUT session.
func (c *Client) Authenticate(username, password string) (bool, error) {
	usernameResp, err := c.SendCommand(fmt.Sprintf("USERNAME %s", username))
	if err != nil {
		return false, err
	}
	passwordResp, err := c.SendCommand(fmt.Sprintf("PASSWORD %s", password))
	if err != nil {
		return false, err
	}
	if usernameResp[0] == "OK" && passwordResp[0] == "OK" {
		return true, nil
	}
	return false, nil
}

// GetUPSList returns a list of all UPSes provided by this NUT instance.
func (c *Client) GetUPSList() ([]UPS, error) {
	upsList := []UPS{}
	resp, err := c.SendCommand("LIST UPS")
	if err != nil {
		return upsList, err
	}
	for _, line := range resp {
		if strings.HasPrefix(line, "UPS ") {
			splitLine := strings.Split(strings.TrimPrefix(line, "UPS "), `"`)
			newUPS, err := NewUPS(strings.TrimSuffix(splitLine[0], " "), c)
			if err != nil {
				return upsList, err
			}
			upsList = append(upsList, newUPS)
		}
	}
	return upsList, err
}

// Help returns a list of the commands supported by NUT.
func (c *Client) Help() (string, error) {
	helpResp, err := c.SendCommand("HELP")
	if err != nil || len(helpResp) < 1 {
		return "", err
	}
	return helpResp[0], err
}

// GetVersion returns the the version of the server currently in use.
func (c *Client) GetVersion() (string, error) {
	versionResponse, err := c.SendCommand("VER")
	if err != nil || len(versionResponse) < 1 {
		return "", err
	}
	c.Version = versionResponse[0]
	return versionResponse[0], err
}

// GetNetworkProtocolVersion returns the version of the network protocol currently in use.
func (c *Client) GetNetworkProtocolVersion() (string, error) {
	versionResponse, err := c.SendCommand("NETVER")
	if err != nil || len(versionResponse) < 1 {
		return "", err
	}
	c.ProtocolVersion = versionResponse[0]
	return versionResponse[0], err
}
