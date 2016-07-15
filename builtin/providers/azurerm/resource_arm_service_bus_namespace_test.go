package azurerm

import (
	"fmt"
	"testing"

	"github.com/Azure/azure-sdk-for-go/arm/servicebus"
	"github.com/Azure/azure-sdk-for-go/core/http"
	"github.com/hashicorp/terraform/helper/acctest"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAzureRMServiceBusNamespaceType_get(t *testing.T) {
	valid := []string{"Messaging", "EventHub", "NotificationHub"}

	for _, s := range valid {
		if _, err := validateServiceBusNamespaceType(s, "type"); err != nil {
			t.Errorf("err should be nil for valid type: %#v", err)
		}
	}

	if _, err := validateServiceBusNamespaceType("InvalidType", "type"); err == nil {
		t.Error("Expected validation error")
	}
}

func TestAzureRMServiceBusNamespaceType_validation(t *testing.T) {
	expected := map[string]servicebus.NamespaceType{
		"Messaging":       servicebus.NamespaceTypeMessaging,
		"EventHub":        servicebus.NamespaceTypeEventHub,
		"NotificationHub": servicebus.NamespaceTypeNotificationHub,
	}

	for k, v := range expected {
		if mapped, err := getServiceBusNamespaceType(k); err != nil {
			t.Errorf("err should be nil for valid type: %#v", err)
		} else if mapped != v {
			t.Errorf("expected %s to map to %s", k, v)
		}
	}
}

func TestAccAzureRMServiceBusNamespace_basic(t *testing.T) {
	ri := acctest.RandInt()
	config := fmt.Sprintf(testAccAzureRMServiceBusNamespace_genericType, ri, ri, "messaging")

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testCheckAzureRMServiceBusNamespaceDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: config,
				Check: resource.ComposeTestCheckFunc(
					testCheckAzureRMServiceBusNamespaceExists("azurerm_service_bus_namespace.test"),
				),
			},
		},
	})
}

func TestAccAzureRMServiceBusNamespace_withTags(t *testing.T) {

	ri := acctest.RandInt()
	preConfig := fmt.Sprintf(testAccAzureRMServiceBusNamespace_withTags, ri, ri)
	postConfig := fmt.Sprintf(testAccAzureRMServiceBusNamespace_withTagsUpdated, ri, ri)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testCheckAzureRMServiceBusNamespaceDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: preConfig,
				Check: resource.ComposeTestCheckFunc(
					testCheckAzureRMServiceBusNamespaceExists("azurerm_service_bus_namespace.test"),
					resource.TestCheckResourceAttr(
						"azurerm_service_bus_namespace.test", "tags.%", "2"),
					resource.TestCheckResourceAttr(
						"azurerm_service_bus_namespace.test", "tags.environment", "Production"),
					resource.TestCheckResourceAttr(
						"azurerm_service_bus_namespace.test", "tags.cost_center", "MSFT"),
				),
			},

			resource.TestStep{
				Config: postConfig,
				Check: resource.ComposeTestCheckFunc(
					testCheckAzureRMServiceBusNamespaceExists("azurerm_service_bus_namespace.test"),
					resource.TestCheckResourceAttr(
						"azurerm_service_bus_namespace.test", "tags.%", "1"),
					resource.TestCheckResourceAttr(
						"azurerm_service_bus_namespace.test", "tags.environment", "staging"),
				),
			},
		},
	})
}

func testCheckAzureRMServiceBusNamespaceExists(name string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		// Ensure we have enough information in state to look up in API
		rs, ok := s.RootModule().Resources[name]
		if !ok {
			return fmt.Errorf("Not found: %s", name)
		}

		namespaceName := rs.Primary.Attributes["name"]
		resourceGroup := rs.Primary.Attributes["resource_group_name"]

		conn := testAccProvider.Meta().(*ArmClient).serviceBusNamespaceClient

		resp, err := conn.Get(resourceGroup, namespaceName)
		if err != nil {
			return fmt.Errorf("Bad: Get on namespace: %s", err)
		}

		if resp.StatusCode == http.StatusNotFound {
			return fmt.Errorf("Bad: namespace %q (resource group: %q) does not exist", name, resourceGroup)
		}

		return nil
	}
}

func testCheckAzureRMServiceBusNamespaceDestroy(s *terraform.State) error {
	conn := testAccProvider.Meta().(*ArmClient).serviceBusNamespaceClient

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "azurerm_service_bus_namespace" {
			continue
		}

		name := rs.Primary.Attributes["name"]
		resourceGroup := rs.Primary.Attributes["resource_group_name"]

		resp, err := conn.Get(resourceGroup, name)
		if err != nil {
			return nil
		}

		if resp.StatusCode != http.StatusNotFound {
			return fmt.Errorf("Service Bus namsepace still exists:\n%#v", resp.Properties)
		}
	}

	return nil
}

var testAccAzureRMServiceBusNamespace_genericType = `
resource "azurerm_resource_group" "test" {
    name = "acctestrg-%d"
    location = "West US"
}

resource "azurerm_service_bus_namespace" "test" {
    name = "acctestsbns%d"
    type = "%s"
    location = "West US"
    resource_group_name = "${azurerm_resource_group.test.name}"
}
`

var testAccAzureRMServiceBusNamespace_withTags = `
resource "azurerm_resource_group" "test" {
    name = "acctestrg-%d"
    location = "West US"
}

resource "azurerm_service_bus_namespace" "test" {
    name = "acctestsbns%d"
    type = "messaging"
    location = "West US"
    resource_group_name = "${azurerm_resource_group.test.name}"
    tags {
        environment = "Production"
        cost_center = "MSFT"
    }
}
`

var testAccAzureRMServiceBusNamespace_withTagsUpdated = `
resource "azurerm_resource_group" "test" {
    name = "acctestrg-%d"
    location = "West US"
}

resource "azurerm_service_bus_namespace" "test" {
    name = "acctestsbns%d"
    type = "messaging"
    location = "West US"
    resource_group_name = "${azurerm_resource_group.test.name}"
    tags {
        environment = "staging"
    }
}
`
