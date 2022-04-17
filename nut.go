// Package nut provides a Golang interface for interacting with Network UPS Tools (NUT).
//
// It communicates with NUT over the TCP protocol
package nut

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"net"
	"strings"
)

// Client contains information about the NUT server as well as the connection.
type Client struct {
	Version         string
	ProtocolVersion string

	conn net.Conn
}

// NewClient accepts a hostname/IP string and an optional port, then creates a connection to NUT, returning a Client.
func NewClient(ctx context.Context, hostname string, port int) (*Client, error) {
	d := &net.Dialer{}

	conn, err := d.DialContext(ctx, "tcp", fmt.Sprintf("%s:%d", hostname, port))
	if err != nil {
		return nil, fmt.Errorf("%w: dial context fail", err)
	}

	client := Client{
		conn: conn,
	}

	client.Version, err = client.GetVersion()
	if err != nil {
		return nil, fmt.Errorf("%w: get version fail", err)
	}

	client.ProtocolVersion, err = client.GetNetworkProtocolVersion()
	if err != nil {
		return nil, fmt.Errorf("%w: get network protocol version fail", err)
	}

	return &client, nil
}

// Disconnect gracefully disconnects from NUT by sending the LOGOUT command.
func (c *Client) Disconnect() (bool, error) {
	logoutResp, err := c.SendCommand("LOGOUT")
	if err != nil {
		return false, fmt.Errorf("%w: send command fail", err)
	}
	if logoutResp[0] == "OK Goodbye" || logoutResp[0] == "Goodbye..." {
		return true, nil
	}

	return false, nil
}

// ReadResponse is a convenience function for reading newline delimited responses.
func (c *Client) ReadResponse(endLine string, multiLineResponse bool) ([]string, error) {
	connbuff := bufio.NewReader(c.conn)

	var response []string

	for {
		line, err := connbuff.ReadString('\n')
		if err != nil && err != io.EOF {
			return nil, fmt.Errorf("%w: error reading response", err)
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

	return response, nil
}

// SendCommand sends the string cmd to the device, and returns the response.
func (c *Client) SendCommand(cmd string) ([]string, error) {
	cmd = fmt.Sprintf("%v\n", cmd)
	endLine := fmt.Sprintf("END %s", cmd)

	if strings.HasPrefix(cmd, "USERNAME ") ||
		strings.HasPrefix(cmd, "PASSWORD ") ||
		strings.HasPrefix(cmd, "SET ") ||
		strings.HasPrefix(cmd, "HELP ") ||
		strings.HasPrefix(cmd, "VER ") ||
		strings.HasPrefix(cmd, "NETVER ") {
		endLine = "OK\n"
	}

	if _, err := fmt.Fprint(c.conn, cmd); err != nil {
		return nil, fmt.Errorf("%w: fprint fail", err)
	}

	resp, err := c.ReadResponse(endLine, strings.HasPrefix(cmd, "LIST "))
	if err != nil {
		return nil, fmt.Errorf("%w: read response fail", err)
	}
	if strings.HasPrefix(resp[0], "ERR ") {
		return nil, fmt.Errorf("%w: NUT error", errorForMessage(strings.Split(resp[0], " ")[1]))
	}

	return resp, nil
}

// Authenticate accepts a username and passwords and uses them to authenticate the existing NUT session.
func (c *Client) Authenticate(username, password string) (bool, error) {
	usernameResp, err := c.SendCommand(fmt.Sprintf("USERNAME %s", username))
	if err != nil {
		return false, fmt.Errorf("%w: send command username fail", err)
	}

	passwordResp, err := c.SendCommand(fmt.Sprintf("PASSWORD %s", password))
	if err != nil {
		return false, fmt.Errorf("%w: send command password fail", err)
	}
	if usernameResp[0] == "OK" && passwordResp[0] == "OK" {
		return true, nil
	}

	return false, nil
}

// GetUPSList returns a list of all UPSes provided by this NUT instance.
func (c *Client) GetUPSList() ([]*UPS, error) {
	resp, err := c.SendCommand("LIST UPS")
	if err != nil {
		return nil, fmt.Errorf("%w: send command ups list fail", err)
	}

	var upsList []*UPS

	for _, line := range resp {
		if strings.HasPrefix(line, "UPS ") {
			splitLine := strings.Split(strings.TrimPrefix(line, "UPS "), `"`)

			ups, err := NewUPS(strings.TrimSuffix(splitLine[0], " "), c)
			if err != nil {
				return nil, fmt.Errorf("%w: prepare ups list fail", err)
			}

			upsList = append(upsList, ups)
		}
	}

	return upsList, nil
}

// Help returns a list of the commands supported by NUT.
func (c *Client) Help() (string, error) {
	helpResp, err := c.SendCommand("HELP")
	if err != nil || len(helpResp) < 1 {
		return "", fmt.Errorf("%w: send command help fail", err)
	}

	return helpResp[0], nil
}

// GetVersion returns the the version of the server currently in use.
func (c *Client) GetVersion() (string, error) {
	versionResponse, err := c.SendCommand("VER")
	if err != nil || len(versionResponse) < 1 {
		return "", fmt.Errorf("%w: send command var fail", err)
	}

	return versionResponse[0], nil
}

// GetNetworkProtocolVersion returns the version of the network protocol currently in use.
func (c *Client) GetNetworkProtocolVersion() (string, error) {
	versionResponse, err := c.SendCommand("NETVER")
	if err != nil || len(versionResponse) < 1 {
		return "", fmt.Errorf("%w: send command netver fail", err)
	}

	return versionResponse[0], nil
}
