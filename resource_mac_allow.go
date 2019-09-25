package main

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"log"
	"net"
	"strings"

	"github.com/hashicorp/terraform/helper/schema"
)

// https://docs.microsoft.com/en-us/powershell/module/dhcpserver/Add-DhcpServerv4Filter?view=win10-ps

func resourceMacAllow() *schema.Resource {
	return &schema.Resource{
		Create: resourceServerCreate,
		Read:   resourceServerRead,
		Update: resourceServerUpdate,
		Delete: resourceServerDelete,

		Schema: map[string]*schema.Schema{
			"mac": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Default:  "",
			},
			"description": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},
			"mac_windows": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},
		},
	}
}

func resourceServerCreate(d *schema.ResourceData, m interface{}) error {

	c := m.(*Communicator)
	c.Connect()

	var mac net.HardwareAddr

	if d.Get("mac") == "" {
		log.Printf("[DEBUG] generate a mac address")
		mac = GenerateMac(c.GetAllAllowedMacAddress())
	}

	var err error

	if d.Get("mac") != "" {
		log.Printf("[DEBUG] validate a mac address")
		mac, err = net.ParseMAC(d.Get("mac").(string))
		if err != nil {
			log.Printf("[DEBUG] invalid mac address")
			return errors.New("Invalid mac Address")
		}
	}

	d.Set("mac_windows", NormalizeMacWindows(mac.String()))

	error := c.AddFilterAllowAddress(d.Get("mac_windows").(string), d.Get("description").(string))
	if error != nil {
		return error
	}
	d.SetId(mac.String())
	return nil
}

func resourceServerRead(d *schema.ResourceData, m interface{}) error {
	return nil
}

func resourceServerUpdate(d *schema.ResourceData, m interface{}) error {
	return resourceServerRead(d, m)
}

func resourceServerDelete(d *schema.ResourceData, m interface{}) error {
	c := m.(*Communicator)
	c.Connect()
	return c.RemoveFilterAllowAddress(d.Get("mac_windows").(string))
}

func randomHex(n int) (string, error) {
	bytes := make([]byte, n)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

func GenerateMac(except []string) net.HardwareAddr {
	mac, _ := net.ParseMAC(subGenerate("00", 5))
	if containsMac(NormalizeMacWindows(mac.String()), except) {
		return GenerateMac(except)
	}
	return mac
}

func containsMac(mac string, macs []string) bool {
	for _, n := range macs {
		if mac == n {
			return true
		}
	}
	return false
}

func subGenerate(before string, count int) string {
	if count == 0 {
		return before
	}
	sub, _ := randomHex(1)
	new := before + ":" + sub
	return subGenerate(new, count-1)
}

func NormalizeMacWindows(mac string) string {
	return strings.ToUpper(strings.Replace(mac, ":", "-", -1))
}
