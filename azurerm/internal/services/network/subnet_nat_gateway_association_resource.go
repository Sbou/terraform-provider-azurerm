package network

import (
	"fmt"
	"log"
	"time"

	"github.com/Azure/azure-sdk-for-go/services/network/mgmt/2020-07-01/network"
	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
	"github.com/terraform-providers/terraform-provider-azurerm/azurerm/helpers/azure"
	"github.com/terraform-providers/terraform-provider-azurerm/azurerm/helpers/tf"
	"github.com/terraform-providers/terraform-provider-azurerm/azurerm/internal/clients"
	"github.com/terraform-providers/terraform-provider-azurerm/azurerm/internal/locks"
	"github.com/terraform-providers/terraform-provider-azurerm/azurerm/internal/services/network/parse"
	"github.com/terraform-providers/terraform-provider-azurerm/azurerm/internal/tf/pluginsdk"
	"github.com/terraform-providers/terraform-provider-azurerm/azurerm/internal/timeouts"
	"github.com/terraform-providers/terraform-provider-azurerm/azurerm/utils"
)

func resourceSubnetNatGatewayAssociation() *schema.Resource {
	return &schema.Resource{
		Create: resourceSubnetNatGatewayAssociationCreate,
		Read:   resourceSubnetNatGatewayAssociationRead,
		Delete: resourceSubnetNatGatewayAssociationDelete,
		// TODO: replace this with an importer which validates the ID during import
		Importer: pluginsdk.DefaultImporter(),
		Timeouts: &schema.ResourceTimeout{
			Create: schema.DefaultTimeout(30 * time.Minute),
			Read:   schema.DefaultTimeout(5 * time.Minute),
			Update: schema.DefaultTimeout(30 * time.Minute),
			Delete: schema.DefaultTimeout(30 * time.Minute),
		},

		Schema: map[string]*schema.Schema{
			"subnet_id": {
				Type:         schema.TypeString,
				Required:     true,
				ForceNew:     true,
				ValidateFunc: azure.ValidateResourceID,
			},

			"nat_gateway_id": {
				Type:         schema.TypeString,
				Required:     true,
				ForceNew:     true,
				ValidateFunc: azure.ValidateResourceID,
			},
		},
	}
}

func resourceSubnetNatGatewayAssociationCreate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*clients.Client).Network.SubnetsClient
	ctx, cancel := timeouts.ForCreate(meta.(*clients.Client).StopContext, d)
	defer cancel()

	log.Printf("[INFO] preparing arguments for Subnet <-> NAT Gateway Association creation.")
	subnetId := d.Get("subnet_id").(string)
	natGatewayId := d.Get("nat_gateway_id").(string)
	parsedSubnetId, err := parse.SubnetID(subnetId)
	if err != nil {
		return err
	}

	subnetName := parsedSubnetId.Name
	virtualNetworkName := parsedSubnetId.VirtualNetworkName
	resourceGroup := parsedSubnetId.ResourceGroup

	parsedGatewayId, err := parse.NatGatewayID(natGatewayId)
	if err != nil {
		return fmt.Errorf("Error parsing NAT gateway id '%s': %+v", natGatewayId, err)
	}

	gatewayName := parsedGatewayId.Name

	locks.ByName(gatewayName, natGatewayResourceName)
	defer locks.UnlockByName(gatewayName, natGatewayResourceName)
	locks.ByName(virtualNetworkName, VirtualNetworkResourceName)
	defer locks.UnlockByName(virtualNetworkName, VirtualNetworkResourceName)
	locks.ByName(subnetName, SubnetResourceName)
	defer locks.UnlockByName(subnetName, SubnetResourceName)

	subnet, err := client.Get(ctx, resourceGroup, virtualNetworkName, subnetName, "")
	if err != nil {
		if utils.ResponseWasNotFound(subnet.Response) {
			return fmt.Errorf("Subnet %q (Virtual Network %q / Resource Group %q) was not found!", subnetName, virtualNetworkName, resourceGroup)
		}
		return fmt.Errorf("Error retrieving Subnet %q (Virtual Network %q / Resource Group %q): %+v", subnetName, virtualNetworkName, resourceGroup, err)
	}

	if props := subnet.SubnetPropertiesFormat; props != nil {
		// check if the resources are imported
		if gateway := props.NatGateway; gateway != nil {
			if gateway.ID != nil && subnet.ID != nil {
				return tf.ImportAsExistsError("azurerm_subnet_nat_gateway_association", *subnet.ID)
			}
		}
		props.NatGateway = &network.SubResource{
			ID: utils.String(natGatewayId),
		}
	}

	future, err := client.CreateOrUpdate(ctx, resourceGroup, virtualNetworkName, subnetName, subnet)
	if err != nil {
		return fmt.Errorf("Error updating NAT Gateway Association for Subnet %q (Virtual Network %q / Resource Group %q): %+v", subnetName, virtualNetworkName, resourceGroup, err)
	}

	if err = future.WaitForCompletionRef(ctx, client.Client); err != nil {
		return fmt.Errorf("Error waiting for completion of NAT Gateway Association for Subnet %q (VN %q / Resource Group %q): %+v", subnetName, virtualNetworkName, resourceGroup, err)
	}

	read, err := client.Get(ctx, resourceGroup, virtualNetworkName, subnetName, "")
	if err != nil {
		return fmt.Errorf("Error retrieving Subnet %q (Virtual Network %q / Resource Group %q): %+v", subnetName, virtualNetworkName, resourceGroup, err)
	}
	d.SetId(*read.ID)

	return resourceSubnetNatGatewayAssociationRead(d, meta)
}

func resourceSubnetNatGatewayAssociationRead(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*clients.Client).Network.SubnetsClient
	ctx, cancel := timeouts.ForRead(meta.(*clients.Client).StopContext, d)
	defer cancel()

	id, err := parse.SubnetID(d.Id())
	if err != nil {
		return err
	}

	resourceGroup := id.ResourceGroup
	virtualNetworkName := id.VirtualNetworkName
	subnetName := id.Name

	subnet, err := client.Get(ctx, resourceGroup, virtualNetworkName, subnetName, "")
	if err != nil {
		if utils.ResponseWasNotFound(subnet.Response) {
			log.Printf("[DEBUG] Subnet %q (Virtual Network %q / Resource Group %q) could not be found - removing from state!", subnetName, virtualNetworkName, resourceGroup)
			d.SetId("")
			return nil
		}
		return fmt.Errorf("Error retrieving Subnet %q (Virtual Network %q / Resource Group %q): %+v", subnetName, virtualNetworkName, resourceGroup, err)
	}

	props := subnet.SubnetPropertiesFormat
	if props == nil {
		return fmt.Errorf("Error: `properties` was nil for Subnet %q (Virtual Network %q / Resource Group %q)", subnetName, virtualNetworkName, resourceGroup)
	}
	natGateway := props.NatGateway
	if natGateway == nil {
		log.Printf("[DEBUG] Subnet %q (Virtual Network %q / Resource Group %q) doesn't have a NAT Gateway - removing from state!", subnetName, virtualNetworkName, resourceGroup)
		d.SetId("")
		return nil
	}

	d.Set("subnet_id", subnet.ID)
	d.Set("nat_gateway_id", natGateway.ID)

	return nil
}

func resourceSubnetNatGatewayAssociationDelete(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*clients.Client).Network.SubnetsClient
	ctx, cancel := timeouts.ForDelete(meta.(*clients.Client).StopContext, d)
	defer cancel()

	id, err := parse.SubnetID(d.Id())
	if err != nil {
		return err
	}

	resourceGroup := id.ResourceGroup
	virtualNetworkName := id.VirtualNetworkName
	subnetName := id.Name

	subnet, err := client.Get(ctx, resourceGroup, virtualNetworkName, subnetName, "")
	if err != nil {
		if utils.ResponseWasNotFound(subnet.Response) {
			log.Printf("[DEBUG] Subnet %q (Virtual Network %q / Resource Group %q) could not be found - removing from state!", subnetName, virtualNetworkName, resourceGroup)
			return nil
		}
		return fmt.Errorf("Error retrieving Subnet %q (Virtual Network %q / Resource Group %q): %+v", subnetName, virtualNetworkName, resourceGroup, err)
	}

	props := subnet.SubnetPropertiesFormat
	if props == nil {
		return fmt.Errorf("`Properties` was nil for Subnet %q (Virtual Network %q / Resource Group %q)", subnetName, virtualNetworkName, resourceGroup)
	}
	if props.NatGateway == nil || props.NatGateway.ID == nil {
		log.Printf("[DEBUG] Subnet %q (Virtual Network %q / Resource Group %q) has no NAT Gateway - removing from state!", subnetName, virtualNetworkName, resourceGroup)
		return nil
	}
	parsedGatewayId, err := azure.ParseAzureResourceID(*props.NatGateway.ID)
	if err != nil {
		return err
	}

	gatewayName := parsedGatewayId.Path["natGateways"]
	locks.ByName(gatewayName, natGatewayResourceName)
	defer locks.UnlockByName(gatewayName, natGatewayResourceName)
	locks.ByName(virtualNetworkName, VirtualNetworkResourceName)
	defer locks.UnlockByName(virtualNetworkName, VirtualNetworkResourceName)

	// ensure we get the latest state
	subnet, err = client.Get(ctx, resourceGroup, virtualNetworkName, subnetName, "")
	if err != nil {
		if utils.ResponseWasNotFound(subnet.Response) {
			log.Printf("[DEBUG] Subnet %q (Virtual Network %q / Resource Group %q) could not be found - removing from state!", subnetName, virtualNetworkName, resourceGroup)
			return nil
		}
		return fmt.Errorf("Error retrieving Subnet %q (Virtual Network %q / Resource Group %q): %+v", subnetName, virtualNetworkName, resourceGroup, err)
	}

	subnet.SubnetPropertiesFormat.NatGateway = nil // remove the nat gateway from subnet

	future, err := client.CreateOrUpdate(ctx, resourceGroup, virtualNetworkName, subnetName, subnet)
	if err != nil {
		return fmt.Errorf("Error removing NAT Gateway Association from Subnet %q (Virtual Network %q / Resource Group %q): %+v", subnetName, virtualNetworkName, resourceGroup, err)
	}

	if err = future.WaitForCompletionRef(ctx, client.Client); err != nil {
		return fmt.Errorf("Error waiting for removal of NAT Gateway Association from Subnet %q (Virtual Network %q / Resource Group %q): %+v", subnetName, virtualNetworkName, resourceGroup, err)
	}

	return nil
}
