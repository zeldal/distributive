package workers

import (
	"fmt"
	"net"
	"reflect"
	"testing"
)

var validHosts = []parameters{
	[]string{"eff.org"},
	[]string{"mozilla.org"},
	[]string{"golang.org"},
}

var invalidHosts = []parameters{
	[]string{"asldkjahserbapsidpuflnaskjdcasd.com"},
	[]string{"aspoiqpweroiqewruqpwioepbpasdfb.net"},
	[]string{"lkjqhwelrjblrjbbrbbbnnzasdflbaj.org"},
}

var validURLs = prefixParameter(validHosts, "http://")
var invalidURLs = prefixParameter(invalidHosts, "http://")
var validHostsWithPort = suffixParameter(validHosts, ":80")
var invalidHostsWithPort = suffixParameter(invalidHosts, ":80")

func TestPort(t *testing.T) {
	t.Parallel()
	losers := []parameters{
		[]string{"49151"}, // reserved
		[]string{"5310"},  // Outlaws (1997 video game)
		[]string{"0"},     // reserved
		[]string{"2302"},  // Halo: Combat Evolved multiplayer
	}
	testInputs(t, port, []parameters{}, losers)
}

func TestInterfaceExists(t *testing.T) {
	t.Parallel()
	testInputs(t, interfaceExists, []parameters{[]string{"lo"}}, names)
}

func TestUp(t *testing.T) {
	t.Parallel()
	testInputs(t, up, []parameters{[]string{"lo"}}, names)
}

func TestIP4(t *testing.T) {
	t.Parallel()
	losers := appendParameter(names, "0.0.0.0")
	testInputs(t, ip4, []parameters{}, losers)
}

func TestIP6(t *testing.T) {
	t.Parallel()
	losers := appendParameter(names, "0000:000:0000:000:0000:0000:000:0000")
	testInputs(t, ip6, []parameters{}, losers)
}

func TestGatewayInterface(t *testing.T) {
	t.Parallel()
	testInputs(t, gatewayInterface, []parameters{}, names)
}

func TestHost(t *testing.T) {
	t.Parallel()
	testInputs(t, host, validHosts, invalidHosts)
}

func TestTCP(t *testing.T) {
	t.Parallel()
	testInputs(t, tcp, validHostsWithPort, invalidHostsWithPort)
}

func TestUDP(t *testing.T) {
	t.Parallel()
	testInputs(t, udp, validHostsWithPort, invalidHostsWithPort)
}

func TestTCPTimeout(t *testing.T) {
	t.Parallel()
	winners := appendParameter(validHostsWithPort, "5s")
	losers := appendParameter(validHostsWithPort, "1µs")
	testInputs(t, tcpTimeout, winners, losers)
}

func TestUDPTimeout(t *testing.T) {
	t.Parallel()
	winners := appendParameter(validHostsWithPort, "5s")
	losers := appendParameter(validHostsWithPort, "1µs")
	testInputs(t, udpTimeout, winners, losers)
}

func TestRoutingTableDestination(t *testing.T) {
	t.Parallel()
	losers := names
	testInputs(t, routingTableDestination, []parameters{}, losers)
}

func TestRoutingTableInterface(t *testing.T) {
	t.Parallel()
	losers := names
	testInputs(t, routingTableInterface, []parameters{}, losers)
}

func TestRoutingTableGateway(t *testing.T) {
	t.Parallel()
	losers := names
	testInputs(t, routingTableGateway, []parameters{}, losers)
}

func TestReponseMatches(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("Skipping tests that query remote servers in short mode")
	} else {
		winners := appendParameter(validURLs, "html")
		losers := appendParameter(validURLs, "asfdjhow012u")
		testInputs(t, responseMatches, winners, losers)
	}
}

func TestReponseMatchesInsecure(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("Skipping tests that query remote servers in short mode")
	} else {
		winners := appendParameter(validURLs, "html")
		losers := appendParameter(validURLs, "asfdjhow012u")
		testInputs(t, responseMatches, winners, losers)
	}
}

func isNil(i interface{}) bool {
	return reflect.ValueOf(i).IsNil()
}

func TestGetARecordAddresses(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("Skipping tests that query remote servers in short mode")
	} else {
		for _, srv := range []string{"8.8.8.8", "8.8.4.4"} {
			for _, hosts := range validHosts {
				host := hosts[0]
				ips := getARecordAddresses(host, srv)
				for _, ip := range ips {
					// check if it gave a valid IP address
					if isNil(net.ParseIP(ip)) {
						msg := "Couldn't parse IP given by DNS server"
						msg += "\n\tDNS server: " + srv
						msg += "\n\tTarget: " + host
						msg += "\n\tResponse: " + fmt.Sprint(ips)
						t.Error(msg)
					}
				}
			}
		}
	}
}

func TestARecord(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("Skipping tests that query remote servers in short mode")
	} else {
		losers := appendParameter(validHosts, "failme")
		testInputs(t, responseMatches, []parameters{}, losers)
	}
}
