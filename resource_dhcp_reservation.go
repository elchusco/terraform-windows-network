package main

import (
	"errors"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"net"
)

func resourceDHCPReservation() *schema.Resource {
	return &schema.Resource{
		Create: CreateDHCPReservation,
		Read:   ReadDHCPReservation,
		Update: UpdateDHCPReservation,
		Delete: DeleteDHCPReservation,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},

		Schema: map[string]*schema.Schema{
			"mac": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},
			"ip": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Default:  nil,
				Computed: true,
			},
			"description": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},
			"scope_id": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},
			"name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},
		},
	}
}

func CreateDHCPReservation(d *schema.ResourceData, m interface{}) error {

	mac, err := net.ParseMAC(d.Get("mac").(string))
	if err != nil {
		return errors.New("Invalid mac Address")
	}

	c := m.(*Communicator)
	c.Connect()

	if d.Get("ip").(string) == "" {
		ip, err := c.GetFreeIp(d.Get("scope_id").(string))
		if err != nil {
			return err
		}
		d.Set("ip", ip.String())
	}

	ipv4 := net.ParseIP(d.Get("ip").(string))
	if ipv4 == nil {
		return errors.New("Invalid ip Address")
	}

	c.AddDHCPReservation(NormalizeMacWindows(mac.String()), ipv4, d.Get("scope_id").(string), d.Get("description").(string), d.Get("name").(string))
	d.SetId(mac.String() + "_" + ipv4.String())
	return nil
}

func DeleteDHCPReservation(d *schema.ResourceData, m interface{}) error {
	c := m.(*Communicator)
	c.Connect()
	c.RemoveDHCPReservation(NormalizeMacWindows(d.Get("mac").(string)), d.Get("scope_id").(string))
	c.RemoveDHCPLease(d.Get("scope_id").(string), d.Get("mac").(string), d.Get("ip").(string))
	return nil
}

func ReadDHCPReservation(d *schema.ResourceData, m interface{}) error {
	return nil
}

func UpdateDHCPReservation(d *schema.ResourceData, m interface{}) error {
	return nil
}
