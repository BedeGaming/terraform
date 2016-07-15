package azurerm

import (
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/Azure/azure-sdk-for-go/arm/servicebus"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceArmServiceBusNamespace() *schema.Resource {
	return &schema.Resource{
		Create: resourceArmServiceBusNamespaceCreate,
		Read:   resourceArmServiceBusNamespaceRead,
		Update: resourceArmServiceBusNamespaceCreate,
		Delete: resourceArmServiceBusNamespaceDelete,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		Schema: map[string]*schema.Schema{
			"name": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"type": {
				Type:         schema.TypeString,
				Required:     true,
				ForceNew:     true,
				ValidateFunc: validateServiceBusNamespaceType,
				StateFunc: func(id interface{}) string {
					return strings.ToLower(id.(string))
				},
			},

			"location": {
				Type:      schema.TypeString,
				Required:  true,
				ForceNew:  true,
				StateFunc: azureRMNormalizeLocation,
			},

			"resource_group_name": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"tags": tagsSchema(),
		},
	}
}

func resourceArmServiceBusNamespaceCreate(d *schema.ResourceData, meta interface{}) error {
	namespacesClient := meta.(*ArmClient).serviceBusNamespaceClient

	log.Printf("[INFO] preparing arguments for Azure ARM service bus namespace creation.")

	name := d.Get("name").(string)
	location := d.Get("location").(string)
	resGroup := d.Get("resource_group_name").(string)
	tags := d.Get("tags").(map[string]interface{})

	props, err := getServiceBusNamespaceProperties(d)
	if err != nil {
		return fmt.Errorf("Error preparing properties: %s", err)
	}

	nsParams := servicebus.NamespaceCreateOrUpdateParameters{
		Location:   &location,
		Properties: props,
		Tags:       expandTags(tags),
	}

	_, err = namespacesClient.CreateOrUpdate(resGroup, name, nsParams, make(chan struct{}))
	if err != nil {
		return err
	}

	read, err := namespacesClient.Get(resGroup, name)
	if err != nil {
		return err
	}
	if read.ID == nil {
		return fmt.Errorf("Cannot read Service Bus Namespace %s (resource group %s) ID", name, resGroup)
	}

	d.SetId(*read.ID)

	return resourceArmServiceBusNamespaceRead(d, meta)
}

func resourceArmServiceBusNamespaceRead(d *schema.ResourceData, meta interface{}) error {
	namespacesClient := meta.(*ArmClient).serviceBusNamespaceClient

	id, err := parseAzureResourceID(d.Id())
	if err != nil {
		return err
	}
	resGroup := id.ResourceGroup
	name := id.Path["namespaces"]

	resp, err := namespacesClient.Get(resGroup, name)
	if resp.StatusCode == http.StatusNotFound {
		d.SetId("")
		return nil
	}
	if err != nil {
		return fmt.Errorf("Error making Read request on Azure Service Bus Namespace %s: %s", name, err)
	}

	// update appropriate values
	d.Set("name", resp.Name)
	// location is retured in the form "West US" from ARM
	d.Set("location", azureRMNormalizeLocation(*resp.Location))

	flattenAndSetTags(d, resp.Tags)

	return nil
}

func resourceArmServiceBusNamespaceDelete(d *schema.ResourceData, meta interface{}) error {
	namespacesClient := meta.(*ArmClient).serviceBusNamespaceClient

	id, err := parseAzureResourceID(d.Id())
	if err != nil {
		return err
	}
	resGroup := id.ResourceGroup
	name := id.Path["namespaces"]

	_, err = namespacesClient.Delete(resGroup, name, make(chan struct{}))

	return err
}

func getServiceBusNamespaceProperties(d *schema.ResourceData) (*servicebus.NamespaceProperties, error) {
	// first; get address space prefixes:
	namespaceType, err := getServiceBusNamespaceType(d.Get("type").(string))
	if err != nil {
		return nil, err
	}

	name := d.Get("name").(string)

	return &servicebus.NamespaceProperties{
		Name:          &name,
		NamespaceType: namespaceType,
	}, nil
}

func getServiceBusNamespaceType(t string) (servicebus.NamespaceType, error) {
	m := map[string]servicebus.NamespaceType{
		"eventhub":        servicebus.NamespaceTypeEventHub,
		"messaging":       servicebus.NamespaceTypeMessaging,
		"notificationhub": servicebus.NamespaceTypeNotificationHub,
	}

	if mapped, ok := m[strings.ToLower(t)]; ok {
		return mapped, nil
	}

	return "", fmt.Errorf("Unknown Service Bus NamespaceType: %s", t)
}

func validateServiceBusNamespaceType(i interface{}, k string) (s []string, errors []error) {
	if _, err := getServiceBusNamespaceType(i.(string)); err != nil {
		errors = append(errors, fmt.Errorf("type must be one of EventHub, Messaging, NotificationHub"))
	}
	return
}
