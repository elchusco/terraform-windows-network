package main

import (
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func resourceRecordCname() *schema.Resource {
	return &schema.Resource{
		Create: createRecordCname,
		Read:   readRecordCname,
		Update: updateRecordCname,
		Delete: deleteRecordCname,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},

		Schema: map[string]*schema.Schema{
			"name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},
			"alias": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},
			"zone": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},
		},
	}
}

func createRecordCname(d *schema.ResourceData, m interface{}) error {

	c := m.(*Communicator)
	c.Connect()

	zone := d.Get("zone").(string)
	name := d.Get("name").(string)
	alias := d.Get("alias").(string)

	err := c.AddDNSRecordCname(zone, alias, name)

	if err != nil {
		return err
	}

	d.SetId("A_z:" + zone + "_n:" + name + "_alias:" + alias)

	return err
}

func deleteRecordCname(d *schema.ResourceData, m interface{}) error {
	c := m.(*Communicator)
	c.Connect()

	alias := d.Get("alias").(string)

	return c.RemoveDNSRecordCname(d.Get("zone").(string), alias, d.Get("name").(string))
}

func readRecordCname(d *schema.ResourceData, m interface{}) error {
	return nil
}

func updateRecordCname(d *schema.ResourceData, m interface{}) error {
	return nil
}
