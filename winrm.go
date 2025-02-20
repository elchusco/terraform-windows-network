package main

import (
	"errors"
	"fmt"
	"log"
	"math/rand"
	"net"
	"regexp"
	"strings"
	"time"

	"github.com/masterzen/winrm"
)

type WinrmError struct {
	codeReturn int
	message    string
	stderr     string
}

func (e *WinrmError) Error() string {
	return fmt.Sprintf("Winrm (%d) - %s", e.codeReturn, e.message)
}

type Communicator struct {
	username string
	password string
	client   *winrm.Client
	endpoint *winrm.Endpoint
}

func (c *Communicator) Connect() error {
	params := winrm.DefaultParameters
	client, err := winrm.NewClientWithParameters(c.endpoint, c.username, c.password, params)
	if err != nil {
		return err
	}
	shell, err := client.CreateShell()
	if err != nil {
		// error here if cannot connect
		return err
	}
	shell.Close()
	c.client = client
	return nil
}

func (c *Communicator) AddFilterAllowAddress(mac string, description string) error {
	command := fmt.Sprintf(
		"Add-DhcpServerv4Filter -List Allow -MacAddress \"%s\" -Description \"%s\" -Force",
		mac,
		description,
	)

	_, stderr, returnCode := c.Execute(command)

	if returnCode != 0 {
		return &WinrmError{returnCode, "Cannot allow mac address in dhcp.", stderr}
	}

	return nil
}

func (c *Communicator) RemoveFilterAllowAddress(mac string) error {
	command := fmt.Sprintf(
		"Remove-DhcpServerv4Filter \"%s\"", mac,
	)

	c.Execute(command)

	return nil
}

func (c *Communicator) GetAllAllowedMacAddress() []string {
	stdout, _, _ := c.Execute("Get-DhcpServerv4Filter -List Allow")
	lines := strings.Split(stdout, "\n")

	var macs []string

	re := regexp.MustCompile(`(([0-9ABCDEF]{2})-?){6,8}`)

	for _, element := range lines {
		matched, _ := regexp.MatchString(`^(([0-9ABCDEF]{2})-?){6,8}`, element)
		if matched {
			mac := string(re.Find([]byte(element)))
			macs = append(macs, mac)
		}
	}
	return macs
}

func (c *Communicator) AddDHCPReservation(mac string, ip net.IP, scopeId string, description string, name string) error {

	command := fmt.Sprintf(
		"Add-DhcpServerv4Reservation -ScopeId %s -Description \"%s\" -IPAddress %s -Name \"%s\" -ClientId %s -Type Dhcp",
		scopeId, description, ip.String(), name, mac,
	)

	_, stderr, returnCode := c.Execute(command)

	if returnCode != 0 {
		return &WinrmError{returnCode, "Cannot add reservation in dhcp server.", stderr}
	}

	return nil
}

func (c *Communicator) RemoveDHCPReservation(mac string, scopeId string) error {

	command := fmt.Sprintf(
		"Remove-DhcpServerv4Reservation -ScopeId %s -ClientId \"%s\"",
		scopeId, mac,
	)

	c.Execute(command)

	return nil
}

func (c *Communicator) RemoveDHCPLease(scopeId string, mac string, ip string) error {
	command := fmt.Sprintf(
		"Remove-DhcpServerv4Lease -ScopeId %s -ClientId \"%s\" -IPAddress %s",
		scopeId, mac, ip,
	)

	c.Execute(command)

	return nil
}

func (c *Communicator) GetFreeIp(scopeId string) (net.IP, error) {
	command := fmt.Sprintf(
		"Get-DhcpServerv4FreeIPAddress -ScopeId %s -NumAddress 1024",
		scopeId,
	)

	stdout, stderr, exitCode := c.Execute(command)

	if exitCode != 0 {
		return nil, &WinrmError{exitCode, "Cannot get a free ip.", stderr}
	}

	ips := strings.Split(stdout, "\n")

	if len(ips) == 0 {
		return nil, errors.New("No ip available in dhcp")
	}

	rand.Seed(time.Now().UnixNano())
	rand.Shuffle(len(ips), func(i, j int) { ips[i], ips[j] = ips[j], ips[i] })

	ipv4String := strings.TrimSpace(ips[0])

	ipv4 := net.ParseIP(ipv4String)

	if ipv4 == nil {
		return nil, errors.New("Canot parse ip " + ipv4String)
	}

	log.Printf("[DEBUG] free ip for scope " + scopeId + " is " + ipv4.String())
	return ipv4, nil
}

func (c *Communicator) AddDNSRecordA(zone string, ip net.IP, name string) error {
	command := fmt.Sprintf(
		"Add-DnsServerResourceRecordA -Name \"%s\" -ZoneName \"%s\" -AllowUpdateAny -IPv4Address \"%s\"",
		name, zone, ip.String(),
	)

	_, stderr, exitCode := c.Execute(command)

	if exitCode != 0 {
		return &WinrmError{exitCode, "Cannot add A record.", stderr}
	}

	return nil
}

func (c *Communicator) AddDNSRecordCname(zone string, alias string, name string) error {
	command := fmt.Sprintf(
		"Add-DnsServerResourceRecordCname -Name \"%s\" -ZoneName \"%s\" -AllowUpdateAny -HostNameAlias \"%s\"",
		name, zone, alias,
	)

	_, stderr, exitCode := c.Execute(command)

	if exitCode != 0 {
		return &WinrmError{exitCode, "Cannot add CNAME record.", stderr}
	}

	return nil
}


func (c *Communicator) RemoveDNSRecordA(zone string, ip net.IP, name string) error {
	command := fmt.Sprintf(
		"Remove-DnsServerResourceRecord -ZoneName \"%s\" -RRType A -Name \"%s\" -RecordData \"%s\" -Force",
		zone, name, ip.String(),
	)
	
	_, stderr, exitCode := c.Execute(command)
	
	if exitCode != 0 {
		return &WinrmError{exitCode, "Cannot remove A record.", stderr}
	}

	return nil
}

func (c *Communicator) RemoveDNSRecordCname(zone string, alias string, name string) error {
	command := fmt.Sprintf(
		"Remove-DnsServerResourceRecord -ZoneName \"%s\" -RRType Cname -Name \"%s\" -RecordData \"%s\" -Force",
		zone, name, alias,
	)
	
	_, stderr, exitCode := c.Execute(command)
	
	if exitCode != 0 {
		return &WinrmError{exitCode, "Cannot remove CNAME record.", stderr}
	}

	return nil
}

func (c *Communicator) AddDNSRecordPTR(zone string, ip net.IP, name string, ptrArr []string, lastByteArr []string) error {

	ptrdomainname := name + "." + zone

	for i, j := 0, len(ptrArr)-1; i < j; i, j = i+1, j-1 {
		ptrArr[i], ptrArr[j] = ptrArr[j], ptrArr[i]
	}

	for i, j := 0, len(lastByteArr)-1; i < j; i, j = i+1, j-1 {
		lastByteArr[i], lastByteArr[j] = lastByteArr[j], lastByteArr[i]
	}

	lastByte := strings.Join(lastByteArr, ".")

	zonename := strings.Join(ptrArr, ".") + ".in-addr.arpa"

	command := fmt.Sprintf(
		"Add-DnsServerResourceRecordPtr -Name \"%s\" -ZoneName \"%s\" -AllowUpdateAny -AgeRecord -PtrDomainName \"%s\"",
		lastByte, zonename, ptrdomainname,
	)

	_, stderr, exitCode := c.Execute(command)

	log.Printf("generated command for Add PTR: " + command)
	log.Printf(stderr)

	if exitCode != 0 {
		return &WinrmError{exitCode, "Cannot add PTR record.", stderr}
	}

	return nil
}

func (c *Communicator) RemoveDNSRecordPTR(ptrArr []string, lastByteArr []string) error {

	for i, j := 0, len(ptrArr)-1; i < j; i, j = i+1, j-1 {
		ptrArr[i], ptrArr[j] = ptrArr[j], ptrArr[i]
	}

	for i, j := 0, len(lastByteArr)-1; i < j; i, j = i+1, j-1 {
		lastByteArr[i], lastByteArr[j] = lastByteArr[j], lastByteArr[i]
	}

	name := strings.Join(lastByteArr, ".")

	zonename := strings.Join(ptrArr, ".") + ".in-addr.arpa"

	command := fmt.Sprintf(
		"Remove-DnsServerResourceRecord -ZoneName \"%s\" -RRType Ptr -Name \"%s\" -Force",
		zonename, name,
	)

	_, stderr, _ := c.Execute(command)

	log.Printf("generated command for Remove PTR: " + command)
	log.Printf(stderr)

	return nil
}

func (c *Communicator) Execute(command string) (string, string, int) {
	stdout, stderr, returnCode, _ := c.client.RunWithString(winrm.Powershell(command), "")
	return stdout, stderr, returnCode
}
