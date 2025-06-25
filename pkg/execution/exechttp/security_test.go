package exechttp

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestSecureDialer(t *testing.T) {
	client := func(dial DialFunc) http.Client {
		return http.Client{
			Timeout: 500 * time.Millisecond,
			Transport: &http.Transport{
				DialContext: dial,
			},
		}
	}

	t.Run("host.docker.internal", func(t *testing.T) {
		t.Run("disabled", func(t *testing.T) {
			c := client(SecureDialer(SecureDialerOpts{
				dial: func(ctx context.Context, network, addr string) (net.Conn, error) {
					panic("should not resolve!")
				},
			}))
			r, err := c.Get("http://host.docker.internal")
			require.Nil(t, r)
			require.NotNil(t, err)
			require.Contains(t, err.Error(), "accessing docker host")

			c = client(SecureDialer(SecureDialerOpts{
				AllowHostDocker: false,
				dial: func(ctx context.Context, network, addr string) (net.Conn, error) {
					panic("should not resolve!")
				},
			}))
			r, err = c.Get("http://host.docker.internal")
			require.Nil(t, r)
			require.NotNil(t, err)
			require.Contains(t, err.Error(), "accessing docker host")
		})

		t.Run("enabled", func(t *testing.T) {
			c := client(SecureDialer(SecureDialerOpts{
				AllowHostDocker: true,
			}))
			_, err := c.Get("http://host.docker.internal")
			if err != nil {
				require.NotContains(t, err.Error(), "accessing docker host")
				return
			}
			require.Nil(t, err)
		})
	})

	t.Run("private ipv4/ipv6", func(t *testing.T) {
		hosts := []string{
			"localhost",
			"fbi.com", // public to 127.0.0.1
			"0.0.0.0",
			"127.0.0.1",
			"10.1.1.1",
			"10.178.5.2",
			"172.16.1.1",
			"192.168.254.1",
			"169.254.1.1",
			"[::1]:443",
		}

		for _, h := range hosts {
			t.Run(fmt.Sprintf("disabled: %s", h), func(t *testing.T) {
				c := client(SecureDialer(SecureDialerOpts{
					log: true,
					dial: func(ctx context.Context, network, addr string) (net.Conn, error) {
						panic("should not resolve!")
					},
				}))
				r, err := c.Get(fmt.Sprintf("http://%s", h))
				require.Nil(t, r)
				require.NotNil(t, err)
				require.Contains(t, err.Error(), "private IP range")

				c = client(SecureDialer(SecureDialerOpts{
					AllowPrivate: false,
					dial: func(ctx context.Context, network, addr string) (net.Conn, error) {
						panic("should not resolve!")
					},
				}))
				r, err = c.Get(fmt.Sprintf("http://%s", h))
				require.Nil(t, r)
				require.NotNil(t, err)
				require.Contains(t, err.Error(), "private IP range")
			})

			t.Run(fmt.Sprintf("enabled: %s", h), func(t *testing.T) {
				c := client(SecureDialer(SecureDialerOpts{
					AllowPrivate: true,
					log:          true,
					dial: func(ctx context.Context, network, addr string) (net.Conn, error) {
						// do nothing.
						return nil, nil
					},
				}))
				_, err := c.Get(fmt.Sprintf("http://%s", h))
				if err != nil {
					require.NotContains(t, err.Error(), "private IP range")
					return
				}
			})
		}
	})

	t.Run("nat64", func(t *testing.T) {
		hosts := []string{
			"[64:ff9b::d8c6:4fc1]:80",
			"[64:ff9b::7f00:0001]:80",
			"[2001:db8:c000:0201::]:80",
			"[2001:db8:aaaa:c000:0002:0100::]:80",
		}

		for _, h := range hosts {

			t.Run(fmt.Sprintf("disabled: %s", h), func(t *testing.T) {
				c := client(SecureDialer(SecureDialerOpts{
					dial: func(ctx context.Context, network, addr string) (net.Conn, error) {
						panic("should not resolve!")
					},
				}))
				r, err := c.Get(fmt.Sprintf("http://%s", h))
				require.Nil(t, r)
				require.NotNil(t, err)
				require.Contains(t, err.Error(), "NAT64 address")

				c = client(SecureDialer(SecureDialerOpts{
					AllowNAT64: false,
					dial: func(ctx context.Context, network, addr string) (net.Conn, error) {
						panic("should not resolve!")
					},
				}))
				r, err = c.Get(fmt.Sprintf("http://%s", h))
				require.Nil(t, r)
				require.NotNil(t, err)
				require.Contains(t, err.Error(), "NAT64 address")
			})

			t.Run(fmt.Sprintf("enabled: %s", h), func(t *testing.T) {
				c := client(SecureDialer(SecureDialerOpts{
					AllowNAT64: true,
					dial: func(ctx context.Context, network, addr string) (net.Conn, error) {
						return nil, nil
					},
				}))
				_, err := c.Get(fmt.Sprintf("http://%s", h))
				if err != nil {
					require.NotContains(t, err.Error(), "NAT64 address")
					return
				}
			})
		}
	})
}
